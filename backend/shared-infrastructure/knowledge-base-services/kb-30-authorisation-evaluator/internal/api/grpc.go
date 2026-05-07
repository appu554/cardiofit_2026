package api

// gRPC surface is intentionally a stub for the MVP. The REST surface
// covers every capability the integration tests require, and adding gRPC
// would pull in protoc/buf scaffolding without product-visible benefit at
// this layer.
//
// TODO(layer3-v1): generate kb_authorisation_evaluator.proto with:
//   service AuthorisationEvaluator {
//     rpc Authorise(AuthoriseRequest) returns (AuthoriseResponse);
//   }
// and wire a gRPC server alongside the REST handler in cmd/server/main.go.
