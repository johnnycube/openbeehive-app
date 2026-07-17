package service

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/johnnycube/openbeehive-app/server/internal/auth"
	wv1 "github.com/johnnycube/openbeehive-app/server/internal/gen/openbeehive/v1"
	"github.com/johnnycube/openbeehive-app/server/internal/gen/openbeehive/v1/openbeehivev1connect"
	"github.com/johnnycube/openbeehive-app/server/internal/storage"
)

type HiveService struct {
	openbeehivev1connect.UnimplementedHiveServiceHandler
	repo storage.HiveRepo
}

func NewHiveService(repo storage.HiveRepo) *HiveService { return &HiveService{repo: repo} }

func hiveToProto(m *storage.Hive) *wv1.Hive {
	return &wv1.Hive{
		Id:             m.ID,
		OrganizationId: m.OrgID,
		ApiaryId:       m.ApiaryID,
		Name:           m.Name,
		Type:           wv1.HiveType(m.Type),
		Status:         wv1.HiveStatus(m.Status),
		Boxes:          m.Boxes,
		ColonyOrigin:   m.ColonyOrigin,
		Note:           m.Note,
		QrCode:         m.QRCode,
		CreatedAt:      timestamppb.New(m.CreatedAt),
		UpdatedAt:      timestamppb.New(m.UpdatedAt),
	}
}

func (s *HiveService) CreateHive(ctx context.Context, req *connect.Request[wv1.CreateHiveRequest]) (*connect.Response[wv1.CreateHiveResponse], error) {
	id, ok := auth.FromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errEmptyName)
	}
	now := time.Now().UTC()
	m := &storage.Hive{
		ID: uuid.NewString(), OrgID: id.OrgID, ApiaryID: req.Msg.ApiaryId, Name: req.Msg.Name,
		Type: int32(req.Msg.Type), Status: int32(wv1.HiveStatus_HIVE_STATUS_ACTIVE),
		Boxes: req.Msg.Boxes, ColonyOrigin: req.Msg.ColonyOrigin, Note: req.Msg.Note,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&wv1.CreateHiveResponse{Hive: hiveToProto(m)}), nil
}

func (s *HiveService) GetHive(ctx context.Context, req *connect.Request[wv1.GetHiveRequest]) (*connect.Response[wv1.GetHiveResponse], error) {
	id, _ := auth.FromContext(ctx)
	m, err := s.repo.Get(ctx, id.OrgID, req.Msg.Id)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&wv1.GetHiveResponse{Hive: hiveToProto(m)}), nil
}

func (s *HiveService) ListHives(ctx context.Context, req *connect.Request[wv1.ListHivesRequest]) (*connect.Response[wv1.ListHivesResponse], error) {
	id, _ := auth.FromContext(ctx)
	limit, offset := pageParams(req.Msg.Page)
	rows, total, err := s.repo.List(ctx, id.OrgID, req.Msg.ApiaryId, limit, offset)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*wv1.Hive, 0, len(rows))
	for i := range rows {
		out = append(out, hiveToProto(&rows[i]))
	}
	return connect.NewResponse(&wv1.ListHivesResponse{Hives: out, Page: &wv1.PageResponse{Total: int32(total)}}), nil
}

func (s *HiveService) UpdateHive(ctx context.Context, req *connect.Request[wv1.UpdateHiveRequest]) (*connect.Response[wv1.UpdateHiveResponse], error) {
	id, _ := auth.FromContext(ctx)
	p := req.Msg.Hive
	if p == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	m, err := s.repo.Get(ctx, id.OrgID, p.Id)
	if err != nil {
		return nil, mapErr(err)
	}
	m.ApiaryID, m.Name = p.ApiaryId, p.Name
	m.Type, m.Status, m.Boxes = int32(p.Type), int32(p.Status), p.Boxes
	m.ColonyOrigin, m.Note, m.QRCode = p.ColonyOrigin, p.Note, p.QrCode
	m.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&wv1.UpdateHiveResponse{Hive: hiveToProto(m)}), nil
}

func (s *HiveService) DeleteHive(ctx context.Context, req *connect.Request[wv1.DeleteHiveRequest]) (*connect.Response[wv1.DeleteHiveResponse], error) {
	id, _ := auth.FromContext(ctx)
	if err := s.repo.Delete(ctx, id.OrgID, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&wv1.DeleteHiveResponse{}), nil
}

func (s *HiveService) RelocateHive(ctx context.Context, req *connect.Request[wv1.RelocateHiveRequest]) (*connect.Response[wv1.RelocateHiveResponse], error) {
	id, _ := auth.FromContext(ctx)
	m, err := s.repo.Get(ctx, id.OrgID, req.Msg.Id)
	if err != nil {
		return nil, mapErr(err)
	}
	m.ApiaryID = req.Msg.TargetApiaryId
	m.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&wv1.RelocateHiveResponse{Hive: hiveToProto(m)}), nil
}
