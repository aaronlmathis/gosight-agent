package logsender

import (
	"context"
	"sync"
	"time"

	"github.com/aaronlmathis/gosight-agent/internal/config"
	grpcconn "github.com/aaronlmathis/gosight-agent/internal/grpc"
	"github.com/aaronlmathis/gosight-agent/internal/protohelper"
	"github.com/aaronlmathis/gosight-shared/model"
	"github.com/aaronlmathis/gosight-shared/proto"
	"github.com/aaronlmathis/gosight-shared/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	goproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// LogSender holds the gRPC client, connection, and stream.
type LogSender struct {
    client proto.LogServiceClient
    cc     *grpc.ClientConn
    stream proto.LogService_SubmitStreamClient
    wg     sync.WaitGroup
    cfg    *config.Config
    ctx    context.Context
}

// NewSender returns immediately and launches the background connection manager.
func NewSender(ctx context.Context, cfg *config.Config) (*LogSender, error) {
    s := &LogSender{ctx: ctx, cfg: cfg}
    go s.manageConnection()
    return s, nil
}

// manageConnection dials & opens the SubmitStream, tears it down on global disconnect,
// and retries with exponential backoff up to totalCap, then fixed-interval.
// in logsender/sender.go
func (s *LogSender) manageConnection() {
    const (
        initial    = 1 * time.Second
        maxBackoff = 10 * time.Second
        totalCap   = 15 * time.Minute
    )

    backoff := initial
    elapsed := time.Duration(0)
    var lastPause time.Time

    for {
        // 1) If a new global pause began, tear down stream AND conn
        pu := grpcconn.GetPauseUntil()
        if pu.After(lastPause) {
            utils.Info("Global disconnect: closing log stream and connection")
            if s.stream != nil {
                _ = s.stream.CloseSend()
            }
            // clear both so we re-dial completely
            s.stream = nil
            s.cc = nil
            backoff = initial
            elapsed = 0
            lastPause = pu
        }

        // 2) Sleep out the pause window
        grpcconn.WaitForResume()

        // 3) Dial (or re-use) a healthy ClientConn
        if s.cc == nil {
            cc, err := grpcconn.GetGRPCConn(s.cfg)
            if err != nil {
                utils.Info("Server offline (dial): retrying in %s", backoff)
                time.Sleep(backoff)
                elapsed += backoff
                if backoff < maxBackoff {
                    backoff *= 2
                }
                if elapsed >= totalCap {
                    backoff = totalCap
                }
                continue
            }
            s.cc = cc
            s.client = proto.NewLogServiceClient(cc)
            // reset backoff on successful dial
            backoff = initial
            elapsed = 0
        }

        // 4) Open the SubmitStream if we don’t already have one
        if s.stream == nil {
            st, err := s.client.SubmitStream(s.ctx)
            if err != nil {
                utils.Info("Server offline (stream): retrying in %s", backoff)
                time.Sleep(backoff)
                elapsed += backoff
                if backoff < maxBackoff {
                    backoff *= 2
                }
                if elapsed >= totalCap {
                    backoff = totalCap
                }
                continue
            }
            s.stream = st
            utils.Info("Log stream connected")
            // reset backoff on successful stream open
            backoff = initial
            elapsed = 0
        }

        // 5) Brief pause before looping to catch disconnects
        time.Sleep(time.Second)
    }
}


// SendLogs marshals the LogPayload and sends it.
// If no active stream, returns Unavailable so your worker backoff kicks in.
func (s *LogSender) SendLogs(payload *model.LogPayload) error {
    if s.stream == nil {
        return status.Error(codes.Unavailable, "no active log stream")
    }
    // --- begin conversion ---
    // Convert LogPayload to proto.LogPayload
    pbLogs := make([]*proto.LogEntry, 0, len(payload.Logs))
    for _, log := range payload.Logs {
        pb := &proto.LogEntry{
            Timestamp: timestamppb.New(log.Timestamp),
            Level:     log.Level,
            Message:   log.Message,
            Source:    log.Source,
            Category:  log.Category,
            Pid:       int32(log.PID),
            Fields:    log.Fields,
            Tags:      log.Tags,
            Meta:      protohelper.ConvertLogMetaToProto(log.Meta),
        }
        pbLogs = append(pbLogs, pb)
    }
    var metaProto *proto.Meta
    if payload.Meta != nil {
        metaProto = protohelper.ConvertMetaToProtoMeta(payload.Meta)
    }
    req := &proto.LogPayload{
        AgentId:    payload.AgentID,
        HostId:     payload.HostID,
        Hostname:   payload.Hostname,
        EndpointId: payload.EndpointID,
        Timestamp:  timestamppb.New(payload.Timestamp),
        Logs:       pbLogs,
        Meta:       metaProto,
    }
    // --- end conversion ---

    // verify marshal
    if _, err := goproto.Marshal(req); err != nil {
        utils.Error("Failed to marshal LogPayload: %v", err)
        return err
    }

    // send
    utils.Info("Sending %d logs to server", len(pbLogs))
    if err := s.stream.Send(req); err != nil {
        utils.Warn("Log stream send failed: %v", err)
        return err
    }
    utils.Debug("Streamed %d logs", len(pbLogs))
    return nil
}

// Close shuts down worker pool and closes the gRPC connection.
func (s *LogSender) Close() error {
    utils.Info("Closing LogSender... waiting for workers")
    s.wg.Wait()
    utils.Info("All LogSender workers finished")
    return s.cc.Close()
}
