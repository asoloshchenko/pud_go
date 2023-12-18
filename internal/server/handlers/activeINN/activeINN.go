package activeINN

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/asoloshchenko/pud_microservice/internal/lib/api/responce"
	"github.com/asoloshchenko/pud_microservice/internal/lib/logger/sl"
	_ "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type ResponseAPI struct {
	Inn    string `json:"inn"`
	Date   string `json:"date,omitempty"`
	Errors struct {
		Inn []string `json:"inn"`
	} `json:"errors,omitempty"`
}

type Request struct {
	INNS []string `json:"inns" validate:"required"`
	// Alias string `json:"alias,omitempty"`
}

type Response struct {
	Status  string   `json:"status"` // Error, Ok
	Error   string   `json:"error,omitempty"`
	ResList []Result `json:"res_list"`
}

type Result struct {
	Inn        string `json:"inn"`
	IsNotExist bool   `json:"is_not_active"`
	Date       string `json:"date,omitempty"` //expire date?
	Error      string `json:"error,omitempty"`
}

func NewCheckINN(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.NewCheckINN"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		err := render.DecodeJSON(r.Body, &req)
		if errors.Is(err, io.EOF) {
			log.Error("Request body is empty")
			render.JSON(w, r, responce.Error("empty request"))
		}

		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.JSON(w, r, responce.Error("failed to decode request"))
			return
		}

		log.Info("request body decoded", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)

			log.Error("invalid request", sl.Err(err))

			render.JSON(w, r, responce.ValidationError(validateErr))

			return
		}

		list_resp := make([]Result, len(req.INNS))

		temp_map := make(map[string][]string)
		temp_map["k"] = []string{"fl"}

		for i, inn := range req.INNS {
			temp_map["inn"] = []string{inn}
			var respApi ResponseAPI

			resp, err := http.PostForm("https://service.nalog.ru/invalid-inn-proc.json", temp_map)

			if err != nil {
				log.Error("Err in requset", sl.Err(err))
				list_resp[i] = Result{
					Error: err.Error(),
					Inn:   inn,
				}
			} else {

				err = render.DecodeJSON(resp.Body, &respApi)

				log.Debug("len errors", slog.Any("len", len(respApi.Errors.Inn)))

				if err != nil {
					log.Error("failed to decode responce body", sl.Err(err), slog.Any("inn", inn))
					list_resp[i] = Result{
						Error: err.Error(),
						Inn:   inn,
					}
				} else if len(respApi.Errors.Inn) != 0 {
					log.Debug("Errors in response", slog.Any("err", respApi.Errors.Inn[0]), slog.Any("inn", inn))
					list_resp[i] = Result{
						Error: respApi.Errors.Inn[0],
						Inn:   inn,
					}
				} else {
					list_resp[i] = Result{
						Inn:        inn,
						Date:       respApi.Date,
						IsNotExist: respApi.Date != "",
					}
				}

			}

		}
		log.Info("Request sent")
		render.JSON(w, r, Response{
			Status:  responce.StatusOk,
			ResList: list_resp,
		})

	}

}
