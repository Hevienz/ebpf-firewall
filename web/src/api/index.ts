import request from './request'
import type { MetricsReport, TargetPage, LinkType, SourcePage } from '../types'

type ResponseData<T> = Promise<T>

export const ping = () => request.get<any, ResponseData<string>>('/ping')
export const getLinkType = () => request.get<any, ResponseData<LinkType>>('/link-type')

export const getMetricsReport = (top: number = 10) => request.get<any, ResponseData<MetricsReport>>('/metrics', { params: { top } })

export const getSources = (page: number = 1, pageSize: number = 20, order: string = 'last_seen_at', sortDir: string = 'desc') =>
	request.get<any, ResponseData<SourcePage>>('/sources', { params: { page, page_size: pageSize, order, sort_dir: sortDir } })

export const getTargets = (
	sid: string,
	page: number = 1,
	pageSize: number = 20,
	order: string = 'last_seen_at',
	sortDir: string = 'desc'
) => request.get<any, ResponseData<TargetPage>>(`/${sid}/targets`, { params: { page, page_size: pageSize, order, sort_dir: sortDir } })

export const getBlackList = () => request.get('/black')
export const addBlack = (targetId: string) => request.post('/black', { target_id: targetId })
export const deleteBlack = (id: string) => request.delete(`/black/${id}`)
