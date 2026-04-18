package forwarder

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/swarm-deploy/cloud-vector/internal/store/contracts"
)

func ForwardRequest(store contracts.Store) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		// Читаем тело запроса
		reqBody, err := io.ReadAll(req.Body)
		if err != nil {
			slog.Error("failed to read req body", slog.Any("err", err))
			http.Error(writer, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer req.Body.Close()

		// Парсим входящий JSON (массив логов)
		var logs []interface{}
		if err = json.Unmarshal(reqBody, &logs); err != nil {
			slog.Error("[proxy][handler] failed to parse JSON", slog.Any("err", err), slog.Any("body", string(reqBody)))

			http.Error(writer, "Bad Request: invalid JSON", http.StatusBadRequest)
			return
		}

		err = store.Push(req.Context(), logs)
		if err != nil {
			slog.ErrorContext(req.Context(), "[proxy] failed to push logs", slog.Any("err", err))
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		writer.WriteHeader(http.StatusOK)
	}
}
