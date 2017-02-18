package api

import (
	"context"
	"strconv"
)

type PatientDetail struct {
	ID            string
	IsStable      bool
	LastUpdate    string
	MainDicomTags map[string]string
	Studies       []string
	Type          string
}

func (a *Api) PatientDetailsSince(ctx context.Context, since, limit int) (result []PatientDetail, err error) {
	err = a.get(ctx, "patients{?since,limit,expand}", map[string]string{
		"since":  strconv.Itoa(since),
		"limit":  strconv.Itoa(limit),
		"expand": "",
	}, &result)
	return result, err
}

func (a *Api) Patient(ctx context.Context, id string) (result PatientDetail, err error) {
	err = a.get(ctx, "patients/{id}", map[string]string{"id": id}, &result)
	return result, err
}
