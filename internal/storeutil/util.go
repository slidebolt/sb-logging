package storeutil

import logging "github.com/slidebolt/sb-logging-sdk"

func Matches(event logging.Event, req logging.ListRequest) bool {
	if !req.Since.IsZero() && event.TS.Before(req.Since) {
		return false
	}
	if !req.Until.IsZero() && event.TS.After(req.Until) {
		return false
	}
	if req.Source != "" && event.Source != req.Source {
		return false
	}
	if req.Kind != "" && event.Kind != req.Kind {
		return false
	}
	if req.Level != "" && event.Level != req.Level {
		return false
	}
	if req.Plugin != "" && event.Plugin != req.Plugin {
		return false
	}
	if req.Device != "" && event.Device != req.Device {
		return false
	}
	if req.Entity != "" && event.Entity != req.Entity {
		return false
	}
	if req.TraceID != "" && event.TraceID != req.TraceID {
		return false
	}
	return true
}

func CloneEvent(event logging.Event) logging.Event {
	cloned := event
	if event.Data != nil {
		cloned.Data = make(map[string]any, len(event.Data))
		for k, v := range event.Data {
			cloned.Data[k] = v
		}
	}
	return cloned
}
