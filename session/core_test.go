package session_test

import (
	"fmt"
	"testing"

	"github.com/hanchuanchuan/goInception/session"
	. "github.com/pingcap/check"
	"golang.org/x/net/context"
)

var _ = Suite(&testInceptionSuite{})

func testInception(t *testing.T) {
	TestingT(t)
}

type testInceptionSuite struct {
}

// TestDisplayWidth 测试列指定长度参数
func (s *testInceptionSuite) TestInception(c *C) {
	core := session.NewInception()
	// cfg:=config.GetGlobalConfig()
	core.LoadOptions(session.SourceOptions{
		Host:     "127.0.0.1",
		Port:     3306,
		User:     "test",
		Password: "test",
	})
	result, err := core.Audit(context.Background(), "create table test.tt1(id int)")
	c.Assert(err, IsNil)

	for _, row := range result {
		fmt.Println(fmt.Sprintf("%#v", row))
	}

}
