package tap

import "github.com/linkerd/linkerd2/testutil"

var (
	expectedT1 = testutil.TapEvent{
		Method:     "POST",
		Authority:  "t1-svc:9090",
		Path:       "/buoyantio.bb.TheService/theFunction",
		HTTPStatus: "200",
		GrpcStatus: "OK",
		TLS:        "true",
		LineCount:  3,
	}

	expectedT2 = testutil.TapEvent{
		Method:     "POST",
		Authority:  "t2-svc:9090",
		Path:       "/buoyantio.bb.TheService/theFunction",
		HTTPStatus: "200",
		GrpcStatus: "Unknown",
		TLS:        "true",
		LineCount:  3,
	}

	expectedT3 = testutil.TapEvent{
		Method:     "POST",
		Authority:  "t3-svc:8080",
		Path:       "/",
		HTTPStatus: "200",
		GrpcStatus: "",
		TLS:        "true",
		LineCount:  3,
	}

	expectedGateway = testutil.TapEvent{
		Method:     "GET",
		Authority:  "gateway-svc:8080",
		Path:       "/",
		HTTPStatus: "500",
		GrpcStatus: "",
		TLS:        "true",
		LineCount:  3,
	}
)

func testTapAppDeploy() {

}

func testTapDeploy() {

}

func testTapDeployContextNs() {

}

func testTapDisabledDeploy() {

}

func testTapSvcCall() {

}

func testTapPod() {

}

func testTapFilterMethod() {

}

func testTapFilterAuthority() {

}
