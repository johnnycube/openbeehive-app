// Package service holds the Connect-RPC handlers. They map between the
// generated protobuf types and the storage models and enforce tenant
// isolation via the identity from the context.
//
// Note: the imports from internal/gen are produced by `make proto`
// (buf generate). All further services (Hive, Queen, Inspection,
// Task, Stats) follow exactly this pattern.
package service

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	wv1 "github.com/johnnycube/openbeehive-app/server/internal/gen/openbeehive/v1"
	"github.com/johnnycube/openbeehive-app/server/internal/gen/openbeehive/v1/openbeehivev1connect"

	"github.com/johnnycube/openbeehive-app/server/internal/auth"
	"github.com/johnnycube/openbeehive-app/server/internal/storage"
)

// Implements the generated openbeehivev1connect.ApiaryServiceHandler interface.
type ApiaryService struct {
	openbeehivev1connect.UnimplementedApiaryServiceHandler
	repo storage.ApiaryRepo
}

func NewApiaryService(repo storage.ApiaryRepo) *ApiaryService {
	return &ApiaryService{repo: repo}
}

func (s *ApiaryService) CreateApiary(
	ctx context.Context, req *connect.Request[wv1.CreateApiaryRequest],
) (*connect.Response[wv1.CreateApiaryResponse], error) {
	id, ok := auth.FromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errEmptyName)
	}
	now := time.Now().UTC()
	m := &storage.Apiary{
		ID:        uuid.NewString(),
		OrgID:     id.OrgID,
		Name:      req.Msg.Name,
		Address:      req.Msg.Address,
		Lat:       req.Msg.Lat,
		Lng:       req.Msg.Lng,
		Note:     req.Msg.Note,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&wv1.CreateApiaryResponse{Apiary: toProto(m, 0)}), nil
}

func (s *ApiaryService) GetApiary(
	ctx context.Context, req *connect.Request[wv1.GetApiaryRequest],
) (*connect.Response[wv1.GetApiaryResponse], error) {
	id, _ := auth.FromContext(ctx)
	m, err := s.repo.Get(ctx, id.OrgID, req.Msg.Id)
	if err != nil {
		return nil, mapErr(err)
	}
	cnt, _ := s.repo.HiveCount(ctx, id.OrgID, m.ID)
	return connect.NewResponse(&wv1.GetApiaryResponse{Apiary: toProto(m, cnt)}), nil
}

func (s *ApiaryService) ListApiaries(
	ctx context.Context, req *connect.Request[wv1.ListApiariesRequest],
) (*connect.Response[wv1.ListApiariesResponse], error) {
	id, _ := auth.FromContext(ctx)
	limit, offset := pageParams(req.Msg.Page)
	rows, total, err := s.repo.List(ctx, id.OrgID, limit, offset)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*wv1.Apiary, 0, len(rows))
	for i := range rows {
		cnt, _ := s.repo.HiveCount(ctx, id.OrgID, rows[i].ID)
		out = append(out, toProto(&rows[i], cnt))
	}
	return connect.NewResponse(&wv1.ListApiariesResponse{
		Apiaries: out,
		Page:      &wv1.PageResponse{Total: int32(total)},
	}), nil
}

func (s *ApiaryService) UpdateApiary(
	ctx context.Context, req *connect.Request[wv1.UpdateApiaryRequest],
) (*connect.Response[wv1.UpdateApiaryResponse], error) {
	id, _ := auth.FromContext(ctx)
	m, err := s.repo.Get(ctx, id.OrgID, req.Msg.Id)
	if err != nil {
		return nil, mapErr(err)
	}
	m.Name, m.Address, m.Lat, m.Lng, m.Note = req.Msg.Name, req.Msg.Address, req.Msg.Lat, req.Msg.Lng, req.Msg.Note
	m.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&wv1.UpdateApiaryResponse{Apiary: toProto(m, 0)}), nil
}

func (s *ApiaryService) DeleteApiary(
	ctx context.Context, req *connect.Request[wv1.DeleteApiaryRequest],
) (*connect.Response[wv1.DeleteApiaryResponse], error) {
	id, _ := auth.FromContext(ctx)
	if err := s.repo.Delete(ctx, id.OrgID, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&wv1.DeleteApiaryResponse{}), nil
}

// --- Helfer ---

func toProto(m *storage.Apiary, hivesAnzahl int) *wv1.Apiary {
	return &wv1.Apiary{
		Id:             m.ID,
		OrganizationId: m.OrgID,
		Name:           m.Name,
		Address:           m.Address,
		Lat:            m.Lat,
		Lng:            m.Lng,
		Note:          m.Note,
		HiveCount:   int32(hivesAnzahl),
		CreatedAt:      timestamppb.New(m.CreatedAt),
		UpdatedAt:      timestamppb.New(m.UpdatedAt),
	}
}

func pageParams(p *wv1.PageRequest) (limit, offset int) {
	if p == nil || p.PageSize <= 0 {
		return 50, 0
	}
	return int(p.PageSize), 0 // add cursor logic here
}

func mapErr(err error) error {
	if err == storage.ErrNotFound {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

type sentinel string

func (e sentinel) Error() string { return string(e) }

var errEmptyName = sentinel("name must not be empty")
