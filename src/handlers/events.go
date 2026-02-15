package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/erkannt/rechenschaftspflicht/services/authentication"
	"github.com/erkannt/rechenschaftspflicht/services/eventstore"
	"github.com/erkannt/rechenschaftspflicht/views"
	"github.com/julienschmidt/httprouter"
)

type EventResponse struct {
	Tag        string  `json:"tag"`
	Comment    string  `json:"comment"`
	Value      string  `json:"value"`
	ValueNum   float64 `json:"valueNum"`
	RecordedAt string  `json:"recordedAt"`
	RecordedBy string  `json:"recordedBy"`
}

func RecordEventFormHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	err := views.LayoutWithNav(views.NewEventForm()).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error rendering layout: %v", err)
		return
	}
}

func RecordEventPostHandler(eventStore eventstore.EventStore, auth authentication.Auth) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		tag := r.FormValue("tag")
		comment := r.FormValue("comment")
		value := r.FormValue("value")

		recordedAt := time.Now().Format(time.RFC3339)
		recordedBy, _ := auth.GetLoggedInUserEmail(r)

		event := eventstore.Event{
			Tag:        tag,
			Comment:    comment,
			Value:      value,
			RecordedAt: recordedAt,
			RecordedBy: recordedBy,
		}

		if err := eventStore.Record(event); err != nil {
			fmt.Printf("failed to record event: %v\n", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		fmt.Printf("Received: %+v\n", event)

		err := views.LayoutWithNav(views.NewEventFormWithSuccessBanner()).Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error rendering layout: %v", err)
			return
		}
	}
}

func AllEventsHandler(eventStore eventstore.EventStore) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		events, err := eventStore.GetAll()
		if err != nil {
			fmt.Printf("failed to retrieve events: %v\n", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		err = views.LayoutWithNav(views.AllEvents(events)).Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error rendering layout: %v", err)
			return
		}
	}
}

func EventsJsonHandler(eventStore eventstore.EventStore) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		events, err := eventStore.GetAll()
		if err != nil {
			fmt.Printf("failed to retrieve events: %v\n", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		var eventResponses []EventResponse
		for _, event := range events {
			if event.Value == "" {
				continue
			}

			valueNum, err := strconv.ParseFloat(event.Value, 64)
			if err != nil {
				continue
			}

			eventResponse := EventResponse{
				Tag:        event.Tag,
				Comment:    event.Comment,
				Value:      event.Value,
				ValueNum:   valueNum,
				RecordedAt: event.RecordedAt,
				RecordedBy: event.RecordedBy,
			}
			eventResponses = append(eventResponses, eventResponse)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(eventResponses); err != nil {
			fmt.Printf("failed to encode events to json: %v\n", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
}

func PlotsHandler(eventStore eventstore.EventStore) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		events, err := eventStore.GetAll()
		if err != nil {
			http.Error(w, "Failed to retrieve events", http.StatusInternalServerError)
			log.Printf("Error retrieving events from event store: %v", err)
			return
		}
		err = views.LayoutWithNav(views.Plots(events)).Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error rendering layout: %v", err)
			return
		}
	}
}
