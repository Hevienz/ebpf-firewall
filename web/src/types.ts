export interface ISummary {
	count: number
	size: number
}

export interface IPacketData {
	time: string
	packet: number
	bytes: number
}

export interface IInputTarget {
	mac: string
	ip: string
	start_time: number
	last_time: number
	summary: ISummary
	port: Record<number, ISummary>
	eth_type: Record<string, ISummary>
	ip_proto: Record<string, ISummary>
}

export interface IInputPacket {
	src_mac: string
	src_ip: string
	summary: ISummary
	country: string
	city: string
	start_time: number
	last_time: number
	target: Record<string, IInputTarget>
}

export interface IConfigBlack {
	mac: string[]
	ipv4: string[]
	ipv6: string[]
}

export interface IBroadcastSummary {
	country_summary: Record<string, ISummary>
	city_summary: Record<string, ISummary>
	eth_type_summary: Record<string, ISummary>
	ip_proto_summary: Record<string, ISummary>
	day_summary: Record<string, ISummary>
	match_summary: Record<string, ISummary>
	dst_port_summary: Record<number, ISummary>
	input_packets: Record<string, IInputPacket>
	black_summary: Record<string, number>
}

export interface IEnhancedPacketInfo {
	src_ip: string
	dst_ip: string
	src_port: number
	dst_port: number
	src_mac: string
	dst_mac: string
	eth_proto: string
	ip_proto: string
	pkt_size: number
	timestamp: number
	country: string
	city: string
	country_code: string
}

export interface ISummaryData {
	// 新增包数
	inc_packet: number
	// 新增字节数
	inc_bytes: number
	// 总包数
	total_packet: number
	// 总字节数
	total_bytes: number
	// 今日总包数
	day_packet: number
	// 今日总字节数
	day_bytes: number
	// 国家
	country: [string, ISummary, number][]
	// 城市
	city: [string, ISummary, number][]
	// 以太网类型
	eth_type: [string, ISummary, number][]
	// 协议类型
	ip_proto: [string, ISummary, number][]
	// 匹配的包
	match: [string, ISummary, number][]
	// 目的端口
	dst_port: [number, ISummary, number][]
	// 黑名单
	black: [string, number][]
	// 输入包
	input: IInputPacket[]
}

export interface IWebSocketMessage<T = unknown> {
	action: string
	id: string
	payload: T
}

export interface IRule {
	rule_name: string
	ip: string[]
	port: (number | string)[]
	mac: string[]
	eth_type: string[]
	ip_protocol: string[]
}

export interface IWebSocketMatchQueryPayload {
	page: number
	page_size: number
	rule_name?: string
	order?: string
	country?: string
	city?: string
	src_mac?: string
	src_ip?: string
	dst_mac?: string
	dst_ip?: string
	eth_type?: string
	ip_proto?: string
	start_time?: number
	end_time?: number
}

export interface IWebSocketChangeBlackListPayload {
	/** 新增还是删除 */
	inc: boolean
	/** 类型 */
	type: 'mac' | 'ipv4' | 'ipv6'
	/** 数据: mac 地址、 ipv4 地址、 ipv6 地址 */
	data: string
}

export interface ISystemResourceUsage {
	cpu: {
		name: string
		physical_cores: number
		logical_cores: number
		usage_per_core: number[]
		total_usage: number
	}
	memory: {
		total: number
		used: number
		available: number
		usage_rate: number
	}
	disk: {
		mount_point: string
		device: string
		fs_type: string
		total: number
		free: number
		used: number
		usage_rate: number
	}[]
	host: {
		hostname: string
		boot_time: number
		os: string
		platform: string
		version: string
	}
	network: {
		connections: {
			family: number
			type: number
			localaddr: { ip: string; port: number }
			remoteaddr: { ip: string; port: number }
			status: string
		}[]
		interfaces: {
			index: number
			mtu: number
			name: string
			hardwareAddr: string
			flags: string[]
			addrs: {
				addr: string
			}[]
		}[]
	}
}
