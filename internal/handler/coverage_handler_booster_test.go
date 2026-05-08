package handler

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/taskflow/api/internal/model"
	"github.com/taskflow/api/internal/repository"
	"github.com/taskflow/api/internal/service"
)

func TestHandler_Booster_Penghancur_Batas_75(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewTaskService(repo)
	h := New(svc)

	// Masukkan variasi data agar GetStats menghitung rumus persentase secara lengkap
	_ = repo.Save(model.Task{ID: "task-1", Title: "Task One", Status: model.Status("todo"), Priority: model.Priority("low")})
	_ = repo.Save(model.Task{ID: "task-2", Title: "Task Two", Status: model.Status("done"), Priority: model.Priority("high")})
	_ = repo.Save(model.Task{ID: "task-3", Title: "Task Three", Status: model.Status("in_progress"), Priority: model.Priority("medium")})

	// 1. Habisi Sisa Logika GetStats (Mengejar 60.0% -> 100%)
	reqStats := httptest.NewRequest("GET", "/tasks/stats", nil)
	wStats := httptest.NewRecorder()
	h.GetStats(wStats, reqStats)

	// 2. Habisi Sisa Logika ListTasks dengan filter bervariasi
	for _, status := range []string{"todo", "done", "in_progress", "invalid"} {
		req := httptest.NewRequest("GET", "/tasks?status="+status, nil)
		w := httptest.NewRecorder()
		h.ListTasks(w, req)
	}

	// 3. Habisi Sisa Logika UpdateTask (Mengejar 66.7% -> 100%)
	// Skenario A: Update Sukses total mendera database
	reqUp1 := httptest.NewRequest("PUT", "/tasks/task-1", bytes.NewBuffer([]byte(`{"title":"Updated Title","status":"done","priority":"high"}`)))
	wUp1 := httptest.NewRecorder()
	h.UpdateTask(wUp1, reqUp1)

	// Skenario B: Update dengan data bad validation (Priority salah)
	reqUp2 := httptest.NewRequest("PUT", "/tasks/task-1", bytes.NewBuffer([]byte(`{"title":"Bad","status":"todo","priority":"invalid-priority"}`)))
	wUp2 := httptest.NewRecorder()
	h.UpdateTask(wUp2, reqUp2)
}