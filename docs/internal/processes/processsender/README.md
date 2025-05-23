<!-- Code generated by gomarkdoc. DO NOT EDIT -->

# processsender

```go
import "github.com/aaronlmathis/gosight-agent/internal/processes/processsender"
```

## Index

- [type ProcessSender](<#ProcessSender>)
  - [func NewSender\(ctx context.Context, cfg \*config.Config\) \(\*ProcessSender, error\)](<#NewSender>)
  - [func \(s \*ProcessSender\) Close\(\) error](<#ProcessSender.Close>)
  - [func \(s \*ProcessSender\) SendSnapshot\(payload \*model.ProcessPayload\) error](<#ProcessSender.SendSnapshot>)
  - [func \(s \*ProcessSender\) StartWorkerPool\(ctx context.Context, queue \<\-chan \*model.ProcessPayload, workerCount int\)](<#ProcessSender.StartWorkerPool>)


<a name="ProcessSender"></a>
## type [ProcessSender](<https://github.com/aaronlmathis/gosight-agent/blob/main/internal/processes/processsender/processsender.go#L51-L58>)

ProcessSender handles streaming process snapshots

```go
type ProcessSender struct {
    // contains filtered or unexported fields
}
```

<a name="NewSender"></a>
### func [NewSender](<https://github.com/aaronlmathis/gosight-agent/blob/main/internal/processes/processsender/processsender.go#L62>)

```go
func NewSender(ctx context.Context, cfg *config.Config) (*ProcessSender, error)
```

NewSender initializes a new ProcessSender and starts the connection manager. It returns immediately and launches the background connection manager.

<a name="ProcessSender.Close"></a>
### func \(\*ProcessSender\) [Close](<https://github.com/aaronlmathis/gosight-agent/blob/main/internal/processes/processsender/processsender.go#L186>)

```go
func (s *ProcessSender) Close() error
```

Close waits for workers then closes the gRPC connection.

<a name="ProcessSender.SendSnapshot"></a>
### func \(\*ProcessSender\) [SendSnapshot](<https://github.com/aaronlmathis/gosight-agent/blob/main/internal/processes/processsender/processsender.go#L140>)

```go
func (s *ProcessSender) SendSnapshot(payload *model.ProcessPayload) error
```

SendSnapshot sends a ProcessPayload; if stream is down, returns Unavailable.

<a name="ProcessSender.StartWorkerPool"></a>
### func \(\*ProcessSender\) [StartWorkerPool](<https://github.com/aaronlmathis/gosight-agent/blob/main/internal/processes/processsender/task.go#L18>)

```go
func (s *ProcessSender) StartWorkerPool(ctx context.Context, queue <-chan *model.ProcessPayload, workerCount int)
```

StartWorkerPool starts a pool of workers to process incoming process payloads. Each worker will attempt to send the payload to the gRPC server. The number of workers is determined by workerCount. Workers exit when the sender’s context is done or the queue is closed.

Generated by [gomarkdoc](<https://github.com/princjef/gomarkdoc>)
