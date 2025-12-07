package handlers

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	gatewayv1 "logistics/gen/go/logistics/gateway/v1"
	reportv1 "logistics/gen/go/logistics/report/v1"
	"logistics/pkg/logger"
	"logistics/services/gateway-svc/internal/clients"
)

// ReportHandler обработчики отчётов
type ReportHandler struct {
	clients *clients.Manager
}

// NewReportHandler создаёт handler
func NewReportHandler(clients *clients.Manager) *ReportHandler {
	return &ReportHandler{clients: clients}
}

func (h *ReportHandler) GenerateReport(
	ctx context.Context,
	req *connect.Request[gatewayv1.GenerateReportRequest],
) (*connect.Response[gatewayv1.GenerateReportResponse], error) {
	msg := req.Msg

	format := reportv1.ReportFormat(msg.Format)
	options := h.convertOptions(msg.Options)

	var resp *reportv1.GenerateFlowReportResponse
	var err error

	switch source := msg.Source.(type) {
	case *gatewayv1.GenerateReportRequest_FlowSource:
		resp, err = h.clients.Report().GenerateFlowReport(ctx, &reportv1.GenerateFlowReportRequest{
			Graph:   source.FlowSource.Graph,
			Result:  source.FlowSource.Result,
			Format:  format,
			Options: options,
		})
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	if err != nil {
		logger.Log.Error("Report generation failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&gatewayv1.GenerateReportResponse{
		Success: resp.Success,
		Report:  h.convertReportInfo(resp.Metadata),
		Content: resp.Content.Data,
	}), nil
}

func (h *ReportHandler) GetReport(
	ctx context.Context,
	req *connect.Request[gatewayv1.GetReportRequest],
) (*connect.Response[gatewayv1.ReportRecord], error) {
	resp, err := h.clients.Report().GetReport(ctx, req.Msg.ReportId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !resp.Success {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}

	return connect.NewResponse(&gatewayv1.ReportRecord{
		ReportId: resp.Metadata.ReportId,
		Info:     h.convertReportInfo(resp.Metadata),
		Content:  resp.Content.Data,
	}), nil
}

func (h *ReportHandler) DownloadReport(
	ctx context.Context,
	req *connect.Request[gatewayv1.DownloadReportRequest],
	stream *connect.ServerStream[gatewayv1.ReportChunk],
) error {
	resp, err := h.clients.Report().GetReport(ctx, req.Msg.ReportId)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	if !resp.Success {
		return connect.NewError(connect.CodeNotFound, nil)
	}

	// Разбиваем на чанки
	data := resp.Content.Data
	chunkSize := 64 * 1024 // 64KB
	totalChunks := (len(data) + chunkSize - 1) / chunkSize

	for i := 0; i < totalChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(data) {
			end = len(data)
		}

		chunk := &gatewayv1.ReportChunk{
			ChunkIndex:  int32(i),
			TotalChunks: int32(totalChunks),
			Data:        data[start:end],
			IsLast:      i == totalChunks-1,
		}

		if err := stream.Send(chunk); err != nil {
			return err
		}
	}

	return nil
}

func (h *ReportHandler) ListReports(
	ctx context.Context,
	req *connect.Request[gatewayv1.ListReportsRequest],
) (*connect.Response[gatewayv1.ListReportsResponse], error) {
	msg := req.Msg

	resp, err := h.clients.Report().ListReports(ctx, &reportv1.ListReportsRequest{
		Limit:         msg.Limit,
		Offset:        msg.Offset,
		ReportType:    reportv1.ReportType(msg.Type),
		Format:        reportv1.ReportFormat(msg.Format),
		CreatedAfter:  msg.CreatedAfter,
		CreatedBefore: msg.CreatedBefore,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	reports := make([]*gatewayv1.ReportInfo, 0, len(resp.Reports))
	for _, r := range resp.Reports {
		reports = append(reports, h.convertReportInfo(r))
	}

	return connect.NewResponse(&gatewayv1.ListReportsResponse{
		Reports:    reports,
		TotalCount: resp.TotalCount,
		HasMore:    resp.HasMore,
	}), nil
}

func (h *ReportHandler) DeleteReport(
	ctx context.Context,
	req *connect.Request[gatewayv1.DeleteReportRequest],
) (*connect.Response[emptypb.Empty], error) {
	resp, err := h.clients.Report().DeleteReport(ctx, req.Msg.ReportId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !resp.Success {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (h *ReportHandler) GetReportFormats(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[gatewayv1.ReportFormatsResponse], error) {
	resp, err := h.clients.Report().GetSupportedFormats(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	formats := make([]*gatewayv1.ReportFormatInfo, 0, len(resp.Formats))
	for _, f := range resp.Formats {
		types := make([]gatewayv1.ReportType, 0, len(f.SupportedReportTypes))
		for _, t := range f.SupportedReportTypes {
			types = append(types, gatewayv1.ReportType(t))
		}
		formats = append(formats, &gatewayv1.ReportFormatInfo{
			Format:               gatewayv1.ReportFormat(f.Format),
			Name:                 f.Name,
			Extension:            f.Extension,
			MimeType:             f.MimeType,
			SupportsCharts:       f.SupportsCharts,
			SupportsStyling:      f.SupportsStyling,
			SupportedReportTypes: types,
		})
	}

	return connect.NewResponse(&gatewayv1.ReportFormatsResponse{
		Formats: formats,
	}), nil
}

func (h *ReportHandler) convertOptions(opts *gatewayv1.ReportOptions) *reportv1.ReportOptions {
	if opts == nil {
		return nil
	}
	return &reportv1.ReportOptions{
		Title:                  opts.Title,
		Description:            opts.Description,
		Author:                 opts.Author,
		Language:               opts.Language,
		Timezone:               opts.Timezone,
		IncludeGraphDetails:    opts.IncludeGraphDetails,
		IncludeEdgeList:        opts.IncludeEdgeList,
		IncludePathDetails:     opts.IncludePathDetails,
		IncludeRecommendations: opts.IncludeRecommendations,
		IncludeCharts:          opts.IncludeCharts,
		CompanyName:            opts.CompanyName,
		LogoUrl:                opts.LogoUrl,
		Theme:                  opts.Theme,
		Currency:               opts.Currency,
		Tags:                   opts.Tags,
		TtlSeconds:             opts.TtlSeconds,
		SaveToStorage:          opts.SaveToStorage,
	}
}

func (h *ReportHandler) convertReportInfo(m *reportv1.ReportMetadata) *gatewayv1.ReportInfo {
	if m == nil {
		return nil
	}
	return &gatewayv1.ReportInfo{
		ReportId:         m.ReportId,
		Title:            m.Title,
		Type:             gatewayv1.ReportType(m.Type),
		Format:           gatewayv1.ReportFormat(m.Format),
		GeneratedAt:      m.GeneratedAt,
		SizeBytes:        m.SizeBytes,
		GenerationTimeMs: m.GenerationTimeMs,
		Filename:         m.Filename,
		ExpiresAt:        m.ExpiresAt,
	}
}
