export type LinkType = 'offload' | 'driver' | 'generic'

// 基础流量指标数据
export interface TrafficMetrics {
	total_packets: number
	total_bytes: number
	first_seen_at?: number
	last_seen_at?: number
}

// 统计数据
export interface Statistics extends TrafficMetrics {
	key: string
}

// 地理位置信息
export interface GeoLocation {
	country: string
	city: string
}

// 协议信息
export interface Protocol {
	eth_type: number // 或使用枚举
	ip_proto: number // 或使用枚举
}

// 源信息
export interface Source {
	src_mac: string
	src_ip: string
	src_port: number
	location: GeoLocation
}

// 源统计信息 (合并 Source 和 TrafficMetrics)
export interface SourceStatisticResult extends Source, TrafficMetrics {
	key: string
	targets: number
}

export interface Destination extends Protocol {
	target_mac: string
	target_ip: string
	target_port: number
}

export interface TargetStatistic extends Destination, TrafficMetrics {
	key: string
}

export enum DimensionKey {
	Country = 'country',
	City = 'city',
	DstPort = 'dst_port',
	EthType = 'eth_type',
	IPProto = 'ip_proto',
	Match = 'match'
}
// 指标报告
export interface MetricsReport {
	total_packets: number
	total_bytes: number
	day: Statistics[]
	dimension: Record<DimensionKey, Statistics[]>
}

export interface SourcePage {
	total: number
	items: SourceStatisticResult[]
}

export interface TargetPage {
	total: number
	items: TargetStatistic[]
}

export interface DeltaMetrics {
	packets: number
	bytes: number
}

export interface IncrementMetrics extends DeltaMetrics {
	time: string
}
