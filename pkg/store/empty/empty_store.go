package empty

import (
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"

	"github.com/rancher/apiserver/pkg/types"
)

type Store struct {
}

func (e *Store) Delete(_ *types.APIRequest, _ *types.APISchema, _ string) (types.APIObject, error) {
	return types.APIObject{}, validation.NotFound
}

func (e *Store) ByID(_ *types.APIRequest, _ *types.APISchema, _ string) (types.APIObject, error) {
	return types.APIObject{}, validation.NotFound
}

func (e *Store) List(_ *types.APIRequest, _ *types.APISchema) (types.APIObjectList, error) {
	return types.APIObjectList{}, validation.NotFound
}

func (e *Store) Create(_ *types.APIRequest, _ *types.APISchema, _ types.APIObject) (types.APIObject, error) {
	return types.APIObject{}, validation.NotFound
}

func (e *Store) Update(_ *types.APIRequest, _ *types.APISchema, _ types.APIObject, _ string) (types.APIObject, error) {
	return types.APIObject{}, validation.NotFound
}

func (e *Store) Watch(_ *types.APIRequest, _ *types.APISchema, _ types.WatchRequest) (chan types.APIEvent, error) {
	return nil, nil
}
