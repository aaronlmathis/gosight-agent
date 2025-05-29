/*
SPDX-License-Identifier: GPL-3.0-or-later

Copyright (C) 2025 Aaron Mathis aaron.mathis@gmail.com

This file is part of GoSight.

GoSight is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

GoSight is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with GoSight. If not, see https://www.gnu.org/licenses/.
*/

// gosight/agent/internal/protohelper/convert.go
// convert.go - converts internal metric payloads to protobuf format for gRPC.

package protohelper

import (
	"github.com/aaronlmathis/gosight-shared/model"
	"github.com/aaronlmathis/gosight-shared/proto"
)

// ConvertLogMetaToProtoMeta translates the internal LogMeta struct into the proto.LogMeta type.
// It preserves all model.LogEntry environment fields needed for traceability.

func ConvertLogMetaToProto(m *model.LogMeta) *proto.LogMeta {
	if m == nil {
		return &proto.LogMeta{}
	}
	return &proto.LogMeta{
		Platform:      m.Platform,
		AppName:       m.AppName,
		AppVersion:    m.AppVersion,
		ContainerId:   m.ContainerID,
		ContainerName: m.ContainerName,
		Unit:          m.Unit,
		Service:       m.Service,
		EventId:       m.EventID,
		User:          m.User,
		Executable:    m.Executable,
		Path:          m.Path,
		Extra:         m.Extra,
	}
}

// ConvertMetaToProtoMeta translates the internal Meta struct into the proto.Meta type.
// It preserves all identity, system, and environment fields needed for traceability.

func ConvertMetaToProtoMeta(m *model.Meta) *proto.Meta {
	if m == nil {
		return nil
	}
	return &proto.Meta{
		Hostname:             m.Hostname,
		IpAddress:            m.IPAddress,
		Os:                   m.OS,
		OsVersion:            m.OSVersion,
		KernelVersion:        m.KernelVersion,
		Architecture:         m.Architecture,
		CloudProvider:        m.CloudProvider,
		Region:               m.Region,
		AvailabilityZone:     m.AvailabilityZone,
		InstanceId:           m.InstanceID,
		InstanceType:         m.InstanceType,
		AccountId:            m.AccountID,
		ProjectId:            m.ProjectID,
		ResourceGroup:        m.ResourceGroup,
		VpcId:                m.VPCID,
		SubnetId:             m.SubnetID,
		ImageId:              m.ImageID,
		ServiceId:            m.ServiceID,
		ContainerId:          m.ContainerID,
		ContainerName:        m.ContainerName,
		PodName:              m.PodName,
		Namespace:            m.Namespace,
		ContainerImageName:   m.ContainerImageName,
		ContainerImageId:     m.ContainerImageID,
		ClusterName:          m.ClusterName,
		NodeName:             m.NodeName,
		Application:          m.Application,
		Environment:          m.Environment,
		Service:              m.Service,
		Version:              m.Version,
		DeploymentId:         m.DeploymentID,
		PublicIp:             m.PublicIP,
		PrivateIp:            m.PrivateIP,
		MacAddress:           m.MACAddress,
		NetworkInterface:     m.NetworkInterface,
		Labels:               m.Labels,
		EndpointId:           m.EndpointID,
		Platform:             m.Platform,
		PlatformFamily:       m.PlatformFamily,
		PlatformVersion:      m.PlatformVersion,
		KernelArchitecture:   m.KernelArchitecture,
		VirtualizationSystem: m.VirtualizationSystem,
		VirtualizationRole:   m.VirtualizationRole,
		HostId:               m.HostID,
		AgentVersion:         m.AgentVersion,
		AgentId:              m.AgentID,
	}
}
