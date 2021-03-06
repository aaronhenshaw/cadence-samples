package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/cadence/testsuite"
)

type UnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestUnitTestSuite(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}

func (s *UnitTestSuite) Test_WorkflowWithMockActivities() {
	env := s.NewTestWorkflowEnvironment()
	env.OnActivity(createExpenseActivity, mock.Anything, mock.Anything).Return(nil).Once()
	env.OnActivity(waitForDecisionActivity, mock.Anything, mock.Anything).Return("APPROVED", nil).Once()
	env.OnActivity(paymentActivity, mock.Anything, mock.Anything).Return(nil).Once()

	env.ExecuteWorkflow(sampleExpenseWorkflow, "test-expense-id")

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
	var workflowResult string
	err := env.GetWorkflowResult(&workflowResult)
	s.NoError(err)
	s.Equal("COMPLETED", workflowResult)
	env.AssertExpectations(s.T())
}

func (s *UnitTestSuite) Test_WorkflowWithMockServer() {
	env := s.NewTestWorkflowEnvironment()

	// setup mock expense server
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/text")
		switch r.URL.Path {
		case "/create":
		case "/registerCallback":
			taskToken := []byte(r.PostFormValue("task_token"))
			// simulate the expense is approved one hour later.
			env.RegisterDelayedCallback(func() {
				env.CompleteActivity(taskToken, "APPROVED", nil)
			}, time.Hour)
		case "/action":
		}
		io.WriteString(w, "SUCCEED")
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// pointing server to test mock
	expenseServerHostPort = server.URL

	env.ExecuteWorkflow(sampleExpenseWorkflow, "test-expense-id")

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
	var workflowResult string
	err := env.GetWorkflowResult(&workflowResult)
	s.NoError(err)
	s.Equal("COMPLETED", workflowResult)
	env.AssertExpectations(s.T())
}
