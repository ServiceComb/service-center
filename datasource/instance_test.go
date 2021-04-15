/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package datasource_test

import (
	"context"
	"github.com/apache/servicecomb-service-center/datasource"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"github.com/apache/servicecomb-service-center/server/config"
	"github.com/apache/servicecomb-service-center/server/core"
	pb "github.com/go-chassis/cari/discovery"
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"testing"
)

func TestInstance_Create(t *testing.T) {
	var serviceId string

	t.Run("create service", func(t *testing.T) {
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				ServiceName: "create_instance_service_ms",
				AppId:       "create_instance_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId = respCreateService.ServiceId
	})

	t.Run("register instance", func(t *testing.T) {
		respCreateInst, err := datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId,
				Endpoints: []string{
					"createInstance_ms:127.0.0.1:8080",
				},
				HostName: "UT-HOST",
				Status:   pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateInst.Response.GetCode())
		assert.NotEqual(t, "", respCreateInst.InstanceId)

		respCreateInst, err = datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				InstanceId: "customId_ms",
				ServiceId:  serviceId,
				Endpoints: []string{
					"createInstance_ms:127.0.0.1:8080",
				},
				HostName: "UT-HOST",
				Status:   pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateInst.Response.GetCode())
		assert.Equal(t, "customId_ms", respCreateInst.InstanceId)
	})

	t.Run("update the same instance", func(t *testing.T) {
		instance := &pb.MicroServiceInstance{
			ServiceId: serviceId,
			Endpoints: []string{
				"sameInstance:127.0.0.1:8080",
			},
			HostName: "UT-HOST",
			Status:   pb.MSI_UP,
		}
		resp, err := datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: instance,
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, resp.Response.GetCode())

		resp, err = datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: instance,
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, resp.Response.GetCode())
		assert.Equal(t, instance.InstanceId, resp.InstanceId)
	})
}

func TestInstance_HeartBeat(t *testing.T) {
	var (
		serviceId   string
		instanceId1 string
		instanceId2 string
	)

	t.Run("register service and instance, should pass", func(t *testing.T) {
		log.Info("register service")
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				ServiceName: "heartbeat_service_ms",
				AppId:       "heartbeat_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId = respCreateService.ServiceId

		respCreateInstance, err := datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"heartbeat:127.0.0.1:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateInstance.Response.GetCode())
		instanceId1 = respCreateInstance.InstanceId

		respCreateInstance, err = datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"heartbeat:127.0.0.2:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateInstance.Response.GetCode())
		instanceId2 = respCreateInstance.InstanceId
	})

	t.Run("update a lease", func(t *testing.T) {
		log.Info("valid instance")
		resp, err := datasource.Instance().Heartbeat(getContext(), &pb.HeartbeatRequest{
			ServiceId:  serviceId,
			InstanceId: instanceId1,
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, resp.Response.GetCode())

		log.Info("serviceId does not exist")
		resp, err = datasource.Instance().Heartbeat(getContext(), &pb.HeartbeatRequest{
			ServiceId:  "100000000000",
			InstanceId: instanceId1,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, pb.ResponseSuccess, resp.Response.GetCode())

		log.Info("instance does not exist")
		resp, err = datasource.Instance().Heartbeat(getContext(), &pb.HeartbeatRequest{
			ServiceId:  serviceId,
			InstanceId: "not-exist-ins",
		})
		assert.NoError(t, err)
		assert.NotEqual(t, pb.ResponseSuccess, resp.Response.GetCode())
	})

	t.Run("batch update lease", func(t *testing.T) {
		log.Info("request contains at least 1 instances")
		resp, err := datasource.Instance().HeartbeatSet(getContext(), &pb.HeartbeatSetRequest{
			Instances: []*pb.HeartbeatSetElement{
				{
					ServiceId:  serviceId,
					InstanceId: instanceId1,
				},
				{
					ServiceId:  serviceId,
					InstanceId: instanceId2,
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, resp.Response.GetCode())
	})
}

func TestInstance_Update(t *testing.T) {
	var (
		serviceId  string
		instanceId string
	)

	t.Run("register service and instance, should pass", func(t *testing.T) {
		log.Info("register service")
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				ServiceName: "update_instance_service_ms",
				AppId:       "update_instance_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId = respCreateService.ServiceId

		log.Info("create instance")
		respCreateInstance, err := datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId,
				Endpoints: []string{
					"updateInstance:127.0.0.1:8080",
				},
				HostName:   "UT-HOST-MS",
				Status:     pb.MSI_UP,
				Properties: map[string]string{"nodeIP": "test"},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateInstance.Response.GetCode())
		instanceId = respCreateInstance.InstanceId
	})

	t.Run("update instance status", func(t *testing.T) {
		log.Info("update instance status to DOWN")
		respUpdateStatus, err := datasource.Instance().UpdateInstanceStatus(getContext(), &pb.UpdateInstanceStatusRequest{
			ServiceId:  serviceId,
			InstanceId: instanceId,
			Status:     pb.MSI_DOWN,
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respUpdateStatus.Response.GetCode())

		log.Info("update instance status to OUTOFSERVICE")
		respUpdateStatus, err = datasource.Instance().UpdateInstanceStatus(getContext(), &pb.UpdateInstanceStatusRequest{
			ServiceId:  serviceId,
			InstanceId: instanceId,
			Status:     pb.MSI_OUTOFSERVICE,
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respUpdateStatus.Response.GetCode())

		log.Info("update instance status to STARTING")
		respUpdateStatus, err = datasource.Instance().UpdateInstanceStatus(getContext(), &pb.UpdateInstanceStatusRequest{
			ServiceId:  serviceId,
			InstanceId: instanceId,
			Status:     pb.MSI_STARTING,
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respUpdateStatus.Response.GetCode())

		log.Info("update instance status to TESTING")
		respUpdateStatus, err = datasource.Instance().UpdateInstanceStatus(getContext(), &pb.UpdateInstanceStatusRequest{
			ServiceId:  serviceId,
			InstanceId: instanceId,
			Status:     pb.MSI_TESTING,
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respUpdateStatus.Response.GetCode())

		log.Info("update instance status to UP")
		respUpdateStatus, err = datasource.Instance().UpdateInstanceStatus(getContext(), &pb.UpdateInstanceStatusRequest{
			ServiceId:  serviceId,
			InstanceId: instanceId,
			Status:     pb.MSI_UP,
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respUpdateStatus.Response.GetCode())

		log.Info("update instance status with a not exist instance")
		respUpdateStatus, err = datasource.Instance().UpdateInstanceStatus(getContext(), &pb.UpdateInstanceStatusRequest{
			ServiceId:  serviceId,
			InstanceId: "notexistins",
			Status:     pb.MSI_STARTING,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, pb.ResponseSuccess, respUpdateStatus.Response.GetCode())
	})

	t.Run("update instance properties", func(t *testing.T) {
		log.Info("update one properties")
		respUpdateProperties, err := datasource.Instance().UpdateInstanceProperties(getContext(),
			&pb.UpdateInstancePropsRequest{
				ServiceId:  serviceId,
				InstanceId: instanceId,
				Properties: map[string]string{
					"test": "test",
				},
			})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respUpdateProperties.Response.GetCode())

		log.Info("all max properties updated")
		size := 1000
		properties := make(map[string]string, size)
		for i := 0; i < size; i++ {
			s := strconv.Itoa(i) + strings.Repeat("x", 253)
			properties[s] = s
		}
		respUpdateProperties, err = datasource.Instance().UpdateInstanceProperties(getContext(),
			&pb.UpdateInstancePropsRequest{
				ServiceId:  serviceId,
				InstanceId: instanceId,
				Properties: properties,
			})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respUpdateProperties.Response.GetCode())

		log.Info("update instance that does not exist")
		respUpdateProperties, err = datasource.Instance().UpdateInstanceProperties(getContext(),
			&pb.UpdateInstancePropsRequest{
				ServiceId:  serviceId,
				InstanceId: "not_exist_ins",
				Properties: map[string]string{
					"test": "test",
				},
			})
		assert.NoError(t, err)
		assert.NotEqual(t, pb.ResponseSuccess, respUpdateProperties.Response.GetCode())

		log.Info("remove properties")
		respUpdateProperties, err = datasource.Instance().UpdateInstanceProperties(getContext(),
			&pb.UpdateInstancePropsRequest{
				ServiceId:  serviceId,
				InstanceId: instanceId,
			})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respUpdateProperties.Response.GetCode())

		log.Info("update service that does not exist")
		respUpdateProperties, err = datasource.Instance().UpdateInstanceProperties(getContext(),
			&pb.UpdateInstancePropsRequest{
				ServiceId:  "not_exist_service",
				InstanceId: instanceId,
				Properties: map[string]string{
					"test": "test",
				},
			})
		assert.NoError(t, err)
		assert.NotEqual(t, pb.ResponseSuccess, respUpdateProperties.Response.GetCode())
	})
}

func TestInstance_Query(t *testing.T) {
	var (
		serviceId1  string
		serviceId2  string
		serviceId3  string
		serviceId4  string
		serviceId5  string
		serviceId6  string
		serviceId7  string
		serviceId8  string
		serviceId9  string
		instanceId1 string
		instanceId2 string
		instanceId4 string
		instanceId5 string
		instanceId8 string
		instanceId9 string
	)

	t.Run("register services and instances for testInstance_query", func(t *testing.T) {
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "query_instance_ms",
				ServiceName: "query_instance_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId1 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "query_instance_ms",
				ServiceName: "query_instance_service_ms",
				Version:     "1.0.5",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId2 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "query_instance_diff_app_ms",
				ServiceName: "query_instance_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId3 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				Environment: pb.ENV_PROD,
				AppId:       "query_instance_ms",
				ServiceName: "query_instance_diff_env_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId4 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				Environment: pb.ENV_PROD,
				AppId:       "default",
				ServiceName: "query_instance_shared_provider_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
				Properties: map[string]string{
					pb.PropAllowCrossApp: "true",
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId5 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(
			util.SetDomainProject(util.CloneContext(getContext()), "user", "user"),
			&pb.CreateServiceRequest{
				Service: &pb.MicroService{
					AppId:       "default",
					ServiceName: "query_instance_diff_domain_consumer_ms",
					Version:     "1.0.0",
					Level:       "FRONT",
					Status:      pb.MS_UP,
				},
			})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId6 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "default",
				ServiceName: "query_instance_shared_consumer_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId7 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "query_instance_ms",
				ServiceName: "query_instance_with_rev_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId8 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "query_instance_ms",
				ServiceName: "batch_query_instance_with_rev_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId9 = respCreateService.ServiceId

		respCreateInstance, err := datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId1,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"find:127.0.0.1:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateInstance.Response.GetCode())
		instanceId1 = respCreateInstance.InstanceId

		respCreateInstance, err = datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId2,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"find:127.0.0.2:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateInstance.Response.GetCode())
		instanceId2 = respCreateInstance.InstanceId

		respCreateInstance, err = datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId4,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"find:127.0.0.4:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateInstance.Response.GetCode())
		instanceId4 = respCreateInstance.InstanceId

		respCreateInstance, err = datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId5,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"find:127.0.0.5:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateInstance.Response.GetCode())
		instanceId5 = respCreateInstance.InstanceId

		respCreateInstance, err = datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId8,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"find:127.0.0.8:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateInstance.Response.GetCode())
		instanceId8 = respCreateInstance.InstanceId

		respCreateInstance, err = datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId9,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"find:127.0.0.9:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateInstance.Response.GetCode())
		instanceId9 = respCreateInstance.InstanceId
	})

	t.Run("query instance", func(t *testing.T) {
		log.Info("find with version rule")
		respFind, err := datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId1,
			AppId:             "query_instance_ms",
			ServiceName:       "query_instance_service_ms",
			VersionRule:       "latest",
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, instanceId2, respFind.Instances[0].InstanceId)

		respFind, err = datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId1,
			AppId:             "query_instance_ms",
			ServiceName:       "query_instance_service_ms",
			VersionRule:       "1.0.0+",
			Tags:              []string{},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, instanceId2, respFind.Instances[0].InstanceId)

		respFind, err = datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId1,
			AppId:             "query_instance_ms",
			ServiceName:       "query_instance_service_ms",
			VersionRule:       "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, instanceId1, respFind.Instances[0].InstanceId)

		respFind, err = datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId1,
			AppId:             "query_instance",
			ServiceName:       "query_instance_service",
			VersionRule:       "0.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ErrServiceNotExists, respFind.Response.GetCode())

		log.Info("find with env")
		respFind, err = datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId4,
			AppId:             "query_instance_ms",
			ServiceName:       "query_instance_diff_env_service_ms",
			VersionRule:       "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, 1, len(respFind.Instances))
		assert.Equal(t, instanceId4, respFind.Instances[0].InstanceId)

		respFind, err = datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			Environment: pb.ENV_PROD,
			AppId:       "query_instance_ms",
			ServiceName: "query_instance_diff_env_service_ms",
			VersionRule: "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, 1, len(respFind.Instances))
		assert.Equal(t, instanceId4, respFind.Instances[0].InstanceId)

		log.Info("find with rev")
		ctx := util.SetContext(getContext(), util.CtxNocache, "")
		respFind, err = datasource.Instance().FindInstances(ctx, &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId8,
			AppId:             "query_instance_ms",
			ServiceName:       "query_instance_with_rev_ms",
			VersionRule:       "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		rev, _ := ctx.Value(util.CtxResponseRevision).(string)
		assert.Equal(t, instanceId8, respFind.Instances[0].InstanceId)
		assert.NotEqual(t, 0, len(rev))

		util.WithRequestRev(ctx, "x")
		respFind, err = datasource.Instance().FindInstances(ctx, &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId8,
			AppId:             "query_instance_ms",
			ServiceName:       "query_instance_with_rev_ms",
			VersionRule:       "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, instanceId8, respFind.Instances[0].InstanceId)
		assert.Equal(t, ctx.Value(util.CtxResponseRevision), rev)

		log.Info("find should return 200 if consumer is diff apps")
		respFind, err = datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId3,
			AppId:             "query_instance_ms",
			ServiceName:       "query_instance_service_ms",
			VersionRule:       "1.0.5",
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, 0, len(respFind.Instances))

		log.Info("provider tag does not exist")
		respFind, err = datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId1,
			AppId:             "query_instance_ms",
			ServiceName:       "query_instance_service_ms",
			VersionRule:       "latest",
			Tags:              []string{"not_exist_tag"},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, 0, len(respFind.Instances))

		log.Info("shared service discovery")
		config.Server.Config.GlobalVisible = "query_instance_shared_provider_ms"
		core.RegisterGlobalServices()
		core.Service.Environment = pb.ENV_PROD
		respFind, err = datasource.Instance().FindInstances(
			util.SetTargetDomainProject(
				util.SetDomainProject(util.CloneContext(getContext()), "user", "user"),
				"default", "default"),
			&pb.FindInstancesRequest{
				ConsumerServiceId: serviceId6,
				AppId:             "default",
				ServiceName:       "query_instance_shared_provider_ms",
				VersionRule:       "1.0.0",
			})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, 1, len(respFind.Instances))
		assert.Equal(t, instanceId5, respFind.Instances[0].InstanceId)

		respFind, err = datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId7,
			AppId:             "default",
			ServiceName:       "query_instance_shared_provider_ms",
			VersionRule:       "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, 1, len(respFind.Instances))
		assert.Equal(t, instanceId5, respFind.Instances[0].InstanceId)

		log.Info("query same domain deps")
		// todo finish ut after implementing GetConsumerDependencies interface

		core.Service.Environment = pb.ENV_DEV
	})

	t.Run("batch query instances", func(t *testing.T) {
		log.Info("find with version rule")
		respFind, err := datasource.Instance().BatchFind(getContext(), &pb.BatchFindInstancesRequest{
			ConsumerServiceId: serviceId1,
			Services: []*pb.FindService{
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "query_instance_service_ms",
						Version:     "latest",
					},
				},
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "query_instance_service_ms",
						Version:     "1.0.0+",
					},
				},
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "query_instance_service_ms",
						Version:     "0.0.0",
					},
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, int64(0), respFind.Services.Updated[0].Index)
		assert.Equal(t, instanceId2, respFind.Services.Updated[0].Instances[0].InstanceId)
		assert.Equal(t, int64(1), respFind.Services.Updated[1].Index)
		assert.Equal(t, instanceId2, respFind.Services.Updated[1].Instances[0].InstanceId)
		assert.Equal(t, int64(2), respFind.Services.Failed[0].Indexes[0])
		assert.Equal(t, pb.ErrServiceNotExists, respFind.Services.Failed[0].Error.Code)

		log.Info("find with env")
		respFind, err = datasource.Instance().BatchFind(getContext(), &pb.BatchFindInstancesRequest{
			ConsumerServiceId: serviceId4,
			Services: []*pb.FindService{
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "query_instance_diff_env_service_ms",
						Version:     "1.0.0",
					},
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, 1, len(respFind.Services.Updated[0].Instances))
		assert.Equal(t, instanceId4, respFind.Services.Updated[0].Instances[0].InstanceId)

		log.Info("find with rev")
		ctx := util.SetContext(getContext(), util.CtxNocache, "")
		respFind, err = datasource.Instance().BatchFind(ctx, &pb.BatchFindInstancesRequest{
			ConsumerServiceId: serviceId8,
			Services: []*pb.FindService{
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "query_instance_with_rev_ms",
						Version:     "1.0.0",
					},
				},
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "batch_query_instance_with_rev_ms",
						Version:     "1.0.0",
					},
				},
			},
			Instances: []*pb.FindInstance{
				{
					Instance: &pb.HeartbeatSetElement{
						ServiceId:  serviceId9,
						InstanceId: instanceId9,
					},
				},
				{
					Instance: &pb.HeartbeatSetElement{
						ServiceId:  serviceId8,
						InstanceId: instanceId8,
					},
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		rev := respFind.Services.Updated[0].Rev
		assert.Equal(t, int64(0), respFind.Services.Updated[0].Index)
		assert.Equal(t, int64(1), respFind.Services.Updated[1].Index)
		assert.Equal(t, instanceId8, respFind.Services.Updated[0].Instances[0].InstanceId)
		assert.Equal(t, instanceId9, respFind.Services.Updated[1].Instances[0].InstanceId)
		assert.NotEqual(t, 0, len(rev))
		instanceRev := respFind.Instances.Updated[0].Rev
		assert.Equal(t, int64(0), respFind.Instances.Updated[0].Index)
		assert.Equal(t, int64(1), respFind.Instances.Updated[1].Index)
		assert.Equal(t, instanceId9, respFind.Instances.Updated[0].Instances[0].InstanceId)
		assert.Equal(t, instanceId8, respFind.Instances.Updated[1].Instances[0].InstanceId)
		assert.NotEqual(t, 0, len(instanceRev))

		respFind, err = datasource.Instance().BatchFind(ctx, &pb.BatchFindInstancesRequest{
			ConsumerServiceId: serviceId8,
			Services: []*pb.FindService{
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "query_instance_with_rev_ms",
						Version:     "1.0.0",
					},
					Rev: "x",
				},
			},
			Instances: []*pb.FindInstance{
				{
					Instance: &pb.HeartbeatSetElement{
						ServiceId:  serviceId9,
						InstanceId: instanceId9,
					},
					Rev: "x",
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, instanceId8, respFind.Services.Updated[0].Instances[0].InstanceId)
		assert.Equal(t, respFind.Services.Updated[0].Rev, rev)
		assert.Equal(t, instanceId9, respFind.Instances.Updated[0].Instances[0].InstanceId)
		assert.Equal(t, instanceRev, respFind.Instances.Updated[0].Rev)

		respFind, err = datasource.Instance().BatchFind(ctx, &pb.BatchFindInstancesRequest{
			ConsumerServiceId: serviceId8,
			Services: []*pb.FindService{
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "query_instance_with_rev_ms",
						Version:     "1.0.0",
					},
					Rev: rev,
				},
			},
			Instances: []*pb.FindInstance{
				{
					Instance: &pb.HeartbeatSetElement{
						ServiceId:  serviceId9,
						InstanceId: instanceId9,
					},
					Rev: instanceRev,
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, int64(0), respFind.Services.NotModified[0])
		assert.Equal(t, int64(0), respFind.Instances.NotModified[0])

		log.Info("find should return 200 even if consumer is diff apps")
		respFind, err = datasource.Instance().BatchFind(getContext(), &pb.BatchFindInstancesRequest{
			ConsumerServiceId: serviceId3,
			Services: []*pb.FindService{
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "query_instance_service_ms",
						Version:     "1.0.5",
					},
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, 0, len(respFind.Services.Updated[0].Instances))

		log.Info("shared service discovery")
		config.Server.Config.GlobalVisible = "query_instance_shared_provider_ms"
		core.RegisterGlobalServices()
		core.Service.Environment = pb.ENV_PROD
		respFind, err = datasource.Instance().BatchFind(
			util.SetTargetDomainProject(
				util.SetDomainProject(util.CloneContext(getContext()), "user", "user"),
				"default", "default"),
			&pb.BatchFindInstancesRequest{
				ConsumerServiceId: serviceId6,
				Services: []*pb.FindService{
					{
						Service: &pb.MicroServiceKey{
							AppId:       "default",
							ServiceName: "query_instance_shared_provider_ms",
							Version:     "1.0.0",
						},
					},
				},
			})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, 1, len(respFind.Services.Updated[0].Instances))
		assert.Equal(t, instanceId5, respFind.Services.Updated[0].Instances[0].InstanceId)

		respFind, err = datasource.Instance().BatchFind(getContext(), &pb.BatchFindInstancesRequest{
			ConsumerServiceId: serviceId7,
			Services: []*pb.FindService{
				{
					Service: &pb.MicroServiceKey{
						AppId:       "default",
						ServiceName: "query_instance_shared_provider_ms",
						Version:     "1.0.0",
					},
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, 1, len(respFind.Services.Updated[0].Instances))
		assert.Equal(t, instanceId5, respFind.Services.Updated[0].Instances[0].InstanceId)

		respFind, err = datasource.Instance().BatchFind(util.SetTargetDomainProject(
			util.SetDomainProject(util.CloneContext(getContext()), "user", "user"),
			"default", "default"),
			&pb.BatchFindInstancesRequest{
				ConsumerServiceId: serviceId6,
				Instances: []*pb.FindInstance{
					{
						Instance: &pb.HeartbeatSetElement{
							ServiceId:  serviceId5,
							InstanceId: instanceId5,
						},
					},
				},
			})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, pb.ErrServiceNotExists, respFind.Instances.Failed[0].Error.Code)

		respFind, err = datasource.Instance().BatchFind(getContext(), &pb.BatchFindInstancesRequest{
			ConsumerServiceId: serviceId7,
			Instances: []*pb.FindInstance{
				{
					Instance: &pb.HeartbeatSetElement{
						ServiceId:  serviceId5,
						InstanceId: instanceId5,
					},
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, 1, len(respFind.Instances.Updated[0].Instances))
		assert.Equal(t, instanceId5, respFind.Instances.Updated[0].Instances[0].InstanceId)

		core.Service.Environment = pb.ENV_DEV
	})

	t.Run("query instances between diff dimensions", func(t *testing.T) {
		log.Info("diff appId")
		UTFunc := func(consumerId string, code int32) {
			respFind, err := datasource.Instance().GetInstances(getContext(), &pb.GetInstancesRequest{
				ConsumerServiceId: consumerId,
				ProviderServiceId: serviceId2,
			})
			assert.NoError(t, err)
			assert.Equal(t, code, respFind.Response.GetCode())
		}

		UTFunc(serviceId3, pb.ErrServiceNotExists)

		UTFunc(serviceId1, pb.ResponseSuccess)

		log.Info("diff env")
		respFind, err := datasource.Instance().GetInstances(getContext(), &pb.GetInstancesRequest{
			ConsumerServiceId: serviceId4,
			ProviderServiceId: serviceId2,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, pb.ResponseSuccess, respFind.Response.GetCode())
	})
}

func TestServicesStatistics_Get(t *testing.T) {
	var ctx context.Context
	var serviceId1 string
	var serviceId2 string

	t.Run("register services and instances for TestServicesStatistics_Get", func(t *testing.T) {

		//service1
		ctx = util.WithNoCache(util.SetDomainProject(context.Background(), "default", "Project1"))
		respCreateService, err := datasource.Instance().RegisterService(ctx, &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "query_statistics1",
				ServiceName: "service1",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId1 = respCreateService.ServiceId

		respCreateInstance, err := datasource.Instance().RegisterInstance(ctx, &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId1,
				HostName:  "UT-HOST-MS1",
				Endpoints: []string{
					"find:127.0.0.1:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateInstance.Response.GetCode())

		respCreateInstance, err = datasource.Instance().RegisterInstance(ctx, &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId1,
				HostName:  "UT-HOST-MS2",
				Endpoints: []string{
					"find:127.0.0.1:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateInstance.Response.GetCode())

		//service2
		ctx = util.WithNoCache(util.SetDomainProject(context.Background(), "DomainTest1", "Project1"))
		respCreateService, err = datasource.Instance().RegisterService(ctx, &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "query_statistics2",
				ServiceName: "service2-DomainTest1",
				Version:     "1.0.5",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId2 = respCreateService.ServiceId

		respCreateInstance, err = datasource.Instance().RegisterInstance(ctx, &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId2,
				HostName:  "UT-HOST-MS1",
				Endpoints: []string{
					"find:127.0.0.1:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateInstance.Response.GetCode())
	})

	t.Run("query services statistics", func(t *testing.T) {
		ctx = util.WithNoCache(util.SetDomainProject(context.Background(), "default", "Project1"))
		log.Info("query services default domain statistics")
		respFind, err := datasource.Instance().GetServicesStatistics(ctx, &pb.GetServicesRequest{})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, int64(2), respFind.Statistics.Instances.CountByDomain)

		log.Info("query services domain statistics")
		ctx = util.WithNoCache(util.SetDomainProject(context.Background(), "DomainTest1", "Project1"))
		respFind, err = datasource.Instance().GetServicesStatistics(ctx, &pb.GetServicesRequest{})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respFind.Response.GetCode())
		assert.Equal(t, int64(1), respFind.Statistics.Instances.CountByDomain)
	})

	t.Run("delete a service ", func(t *testing.T) {
		ctx = util.WithNoCache(util.SetDomainProject(context.Background(), "default", "Project1"))
		resp, err := datasource.Instance().UnregisterService(ctx, &pb.DeleteServiceRequest{
			ServiceId: serviceId1,
			Force:     false,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, pb.ResponseSuccess, resp.Response.GetCode())

		ctx = util.WithNoCache(util.SetDomainProject(context.Background(), "DomainTest1", "Project1"))
		resp, err = datasource.Instance().UnregisterService(ctx, &pb.DeleteServiceRequest{
			ServiceId: serviceId2,
			Force:     false,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, pb.ResponseSuccess, resp.Response.GetCode())
	})
}

func TestInstance_GetOne(t *testing.T) {
	var (
		serviceId1  string
		serviceId2  string
		serviceId3  string
		instanceId2 string
	)

	t.Run("register service and instances", func(t *testing.T) {
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "get_instance_ms",
				ServiceName: "get_instance_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId1 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "get_instance_ms",
				ServiceName: "get_instance_service_ms",
				Version:     "1.0.5",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId2 = respCreateService.ServiceId

		respCreateInstance, err := datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId2,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"get:127.0.0.2:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		instanceId2 = respCreateInstance.InstanceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "get_instance_cross_ms",
				ServiceName: "get_instance_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId3 = respCreateService.ServiceId
	})

	t.Run("get one instance when invalid request", func(t *testing.T) {
		log.Info("find service itself")
		resp, err := datasource.Instance().GetInstance(getContext(), &pb.GetOneInstanceRequest{
			ConsumerServiceId:  serviceId2,
			ProviderServiceId:  serviceId2,
			ProviderInstanceId: instanceId2,
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, resp.Response.GetCode())

		log.Info("consumer does not exist")
		resp, err = datasource.Instance().GetInstance(getContext(), &pb.GetOneInstanceRequest{
			ConsumerServiceId:  "not-exist-id-ms",
			ProviderServiceId:  serviceId2,
			ProviderInstanceId: instanceId2,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, pb.ResponseSuccess, resp.Response.GetCode())
	})

	t.Run("get between diff apps", func(t *testing.T) {
		resp, err := datasource.Instance().GetInstance(getContext(), &pb.GetOneInstanceRequest{
			ConsumerServiceId:  serviceId3,
			ProviderServiceId:  serviceId2,
			ProviderInstanceId: instanceId2,
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ErrInstanceNotExists, resp.Response.GetCode())

		respAll, err := datasource.Instance().GetInstances(getContext(), &pb.GetInstancesRequest{
			ConsumerServiceId: serviceId3,
			ProviderServiceId: serviceId2,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, pb.ResponseSuccess, respAll.Response.GetCode())
	})

	t.Run("get instances when request is invalid", func(t *testing.T) {
		log.Info("consumer does not exist")
		resp, err := datasource.Instance().GetInstances(getContext(), &pb.GetInstancesRequest{
			ConsumerServiceId: "not-exist-service-ms",
			ProviderServiceId: serviceId2,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, pb.ResponseSuccess, resp.Response.GetCode())

		log.Info("consumer does not exist")
		resp, err = datasource.Instance().GetInstances(getContext(), &pb.GetInstancesRequest{
			ConsumerServiceId: serviceId1,
			ProviderServiceId: serviceId2,
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, resp.Response.GetCode())
	})
}

func TestInstance_GetAll(t *testing.T) {

	t.Run("register 2 instances, get all instances count should return 2", func(t *testing.T) {
		var (
			serviceId1 string
			serviceId2 string
		)
		ctx := util.WithNoCache(util.SetDomainProject(context.Background(), "TestInstance_GetAll", "1"))
		respCreateService, err := datasource.Instance().RegisterService(ctx, &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "get_instance_ms",
				ServiceName: "get_instance_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId1 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(ctx, &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "get_instance_ms",
				ServiceName: "get_instance_service_ms",
				Version:     "1.0.5",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId2 = respCreateService.ServiceId

		respCreateInstance, err := datasource.Instance().RegisterInstance(ctx, &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId1,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"get:127.0.0.2:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateInstance.Response.GetCode())

		respCreateInstance, err = datasource.Instance().RegisterInstance(ctx, &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId2,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"get:127.0.0.3:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateInstance.Response.GetCode())

		respAll, err := datasource.Instance().GetAllInstances(ctx, &pb.GetAllInstancesRequest{})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respAll.Response.GetCode())
		assert.Equal(t, 2, len(respAll.Instances))
	})

	t.Run("domain contain no instances, get all instances should be pass, return 0 instance", func(t *testing.T) {
		ctx := util.WithNoCache(util.SetDomainProject(context.Background(), "TestInstance_GetAll", "2"))
		respAll, err := datasource.Instance().GetAllInstances(ctx, &pb.GetAllInstancesRequest{})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respAll.Response.GetCode())
		assert.Equal(t, 0, len(respAll.Instances))
	})
}

func TestInstance_Unregister(t *testing.T) {
	var (
		serviceId  string
		instanceId string
	)

	t.Run("register service and instances", func(t *testing.T) {
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "unregister_instance_ms",
				ServiceName: "unregister_instance_service_ms",
				Version:     "1.0.5",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
			Tags: map[string]string{
				"test": "test",
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateService.Response.GetCode())
		serviceId = respCreateService.ServiceId

		respCreateInstance, err := datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"unregister:127.0.0.2:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, respCreateInstance.Response.GetCode())
		instanceId = respCreateInstance.InstanceId
	})

	t.Run("unregister instance", func(t *testing.T) {
		resp, err := datasource.Instance().UnregisterInstance(getContext(), &pb.UnregisterInstanceRequest{
			ServiceId:  serviceId,
			InstanceId: instanceId,
		})
		assert.NoError(t, err)
		assert.Equal(t, pb.ResponseSuccess, resp.Response.GetCode())
	})

	t.Run("unregister instance when request is invalid", func(t *testing.T) {
		log.Info("service id does not exist")
		resp, err := datasource.Instance().UnregisterInstance(getContext(), &pb.UnregisterInstanceRequest{
			ServiceId:  "not-exist-id-ms",
			InstanceId: instanceId,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, pb.ResponseSuccess, resp.Response.GetCode())

		log.Info("instance id does not exist")
		resp, err = datasource.Instance().UnregisterInstance(getContext(), &pb.UnregisterInstanceRequest{
			ServiceId:  serviceId,
			InstanceId: "not-exist-id-ms",
		})
		assert.NoError(t, err)
		assert.NotEqual(t, pb.ResponseSuccess, resp.Response.GetCode())
	})
}
