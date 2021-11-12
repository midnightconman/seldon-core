// Copyright 2020 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package resources

import (
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	http "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
)

const (
	RouteConfigurationName = "listener_0"
)

func MakeCluster(clusterName string, eps []Endpoint, isGrpc bool) *cluster.Cluster {
	if isGrpc {
		httpProtocolOptions := http.HttpProtocolOptions{
			UpstreamProtocolOptions: &http.HttpProtocolOptions_ExplicitHttpConfig_{
				ExplicitHttpConfig: &http.HttpProtocolOptions_ExplicitHttpConfig{
					ProtocolConfig: &http.HttpProtocolOptions_ExplicitHttpConfig_Http2ProtocolOptions{
						Http2ProtocolOptions: &core.Http2ProtocolOptions{},
					},
				},
			},
		}
		hpoMarshalled, err := anypb.New(&httpProtocolOptions)
		if err != nil {
			panic(err)
		}
		return &cluster.Cluster{
			Name:                 clusterName,
			ConnectTimeout:       durationpb.New(5 * time.Second),
			ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_STRICT_DNS},
			LbPolicy:             cluster.Cluster_ROUND_ROBIN,
			LoadAssignment:  MakeEndpoint(clusterName, eps),
			DnsLookupFamily:  cluster.Cluster_V4_ONLY,
			TypedExtensionProtocolOptions: map[string]*anypb.Any{"envoy.extensions.upstreams.http.v3.HttpProtocolOptions":hpoMarshalled},
		}
	}
	return &cluster.Cluster{
		Name:                 clusterName,
		ConnectTimeout:       durationpb.New(5 * time.Second),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_STRICT_DNS},
		LbPolicy:             cluster.Cluster_ROUND_ROBIN,
		LoadAssignment:  MakeEndpoint(clusterName, eps),
		DnsLookupFamily:  cluster.Cluster_V4_ONLY,
	}
}

func makeEDSCluster() *cluster.Cluster_EdsClusterConfig {
	return &cluster.Cluster_EdsClusterConfig{
		EdsConfig: makeConfigSource(),
	}
}

func MakeEndpoint(clusterName string, eps []Endpoint) *endpoint.ClusterLoadAssignment {
	var endpoints []*endpoint.LbEndpoint

	for _, e := range eps {
		endpoints = append(endpoints, &endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: &core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Protocol: core.SocketAddress_TCP,
								Address:  e.UpstreamHost,
								PortSpecifier: &core.SocketAddress_PortValue{
									PortValue: e.UpstreamPort,
								},
							},
						},
					},
				},
			},
		})
	}

	return &endpoint.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			LbEndpoints: endpoints,
		}},
	}
}

func MakeRoute(routes []Route) *route.RouteConfiguration {
	var rts []*route.Route

	for _, r := range routes {
		rts = append(rts, &route.Route{
			Name: r.Name+"_http", // Seems optional
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Prefix{
					Prefix: "/v2",
				},
				Headers: []*route.HeaderMatcher{
					{
						Name: "Host", // Header name we will match on
						HeaderMatchSpecifier: &route.HeaderMatcher_ExactMatch{
							ExactMatch: r.Host,
						},
						//TODO: https://github.com/envoyproxy/envoy/blob/c75c1410c8682cb44c9136ce4ad01e6a58e16e8e/api/envoy/api/v2/route/route_components.proto#L1513
						//HeaderMatchSpecifier: &route.HeaderMatcher_StringMatch{
						//	StringMatch: &matcher.StringMatcher{
						//		MatchPattern: &matcher.StringMatcher_Exact{
						//			Exact: r.Host,
						//		},
						//	},
						//},
					},
				},
			},
			Action: &route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_Cluster{
						Cluster: r.HttpCluster,
					},
				},
			},
		})
		rts = append(rts, &route.Route{
			Name: r.Name+"_grpc", // Seems optional
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Prefix{
					Prefix: "/inference.GRPCInferenceService",
				},
				Headers: []*route.HeaderMatcher{
					{
						Name: "Seldon", // Header name we will match on
						HeaderMatchSpecifier: &route.HeaderMatcher_ExactMatch{
							ExactMatch: r.Host,
						},
						//TODO: https://github.com/envoyproxy/envoy/blob/c75c1410c8682cb44c9136ce4ad01e6a58e16e8e/api/envoy/api/v2/route/route_components.proto#L1513
						//HeaderMatchSpecifier: &route.HeaderMatcher_StringMatch{
						//	StringMatch: &matcher.StringMatcher{
						//		MatchPattern: &matcher.StringMatcher_Exact{
						//			Exact: r.Host,
						//		},
						//	},
						//},
					},
				},
			},
			Action: &route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_Cluster{
						Cluster: r.GrpcCluster,
					},
				},
			},
		})
	}

	return &route.RouteConfiguration{
		Name: RouteConfigurationName,
		VirtualHosts: []*route.VirtualHost{{
			Name:    "seldon_service",
			Domains: []string{"*"},
			Routes:  rts,
		}},
	}
}

func MakeHTTPListener(listenerName, address string, port uint32) *listener.Listener {
	// HTTP filter configuration
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource:    makeConfigSource(),
				RouteConfigName: RouteConfigurationName,
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
	}
	pbst, err := anypb.New(manager)
	if err != nil {
		panic(err)
	}

	return &listener.Listener{
		Name: listenerName,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  address,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: port,
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &listener.Filter_TypedConfig{
					TypedConfig: pbst,
				},
			}},
		}},
	}
}

func makeConfigSource() *core.ConfigSource {
	source := &core.ConfigSource{}
	source.ResourceApiVersion = resource.DefaultAPIVersion
	source.ConfigSourceSpecifier = &core.ConfigSource_ApiConfigSource{
		ApiConfigSource: &core.ApiConfigSource{
			TransportApiVersion:       resource.DefaultAPIVersion,
			ApiType:                   core.ApiConfigSource_DELTA_GRPC,
			SetNodeOnFirstMessageOnly: true,
			GrpcServices: []*core.GrpcService{{
				TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: "xds_cluster"},
				},
			}},
		},
	}
	return source
}
