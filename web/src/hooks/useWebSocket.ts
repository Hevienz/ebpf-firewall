import { useState, useEffect, useCallback, useRef } from 'react'
import { date_format } from '../utils'
import {
	IBroadcastSummary,
	ISummary,
	ISummaryData,
	IWebSocketMessage,
	IRule,
	IWebSocketMatchQueryPayload,
	IPacketData,
	IWebSocketChangeBlackListPayload,
	IEnhancedPacketInfo,
	IConfigBlack,
	ISystemResourceUsage
} from '../types'

function useWebSocket(url: string) {
	const [socket, setSocket] = useState<WebSocket | null>(null)
	const [isConnected, setIsConnected] = useState(false)
	const [userClose, setUserClose] = useState(false)
	const [incPacketHistory, setIncPacketHistory] = useState<IPacketData[]>([])
	const [dayHistory, setDayHistory] = useState<IPacketData[]>([])
	const [summaryData, setSummaryData] = useState<ISummaryData>({
		inc_packet: 0,
		inc_bytes: 0,
		total_packet: 0,
		total_bytes: 0,
		day_packet: 0,
		day_bytes: 0,
		country: [],
		city: [],
		eth_type: [],
		ip_proto: [],
		match: [],
		dst_port: [],
		black: [],
		input: []
	})
	const lastTotalBytesRef = useRef(0)
	const lastTotalPacketRef = useRef(0)
	const callbackMap = useRef<Record<string, { create_time: number; resolve: (data: unknown) => void; reject: (error: Error) => void }>>(
		{}
	)

	const parseSummaryItem = (summary: IBroadcastSummary, key: string): [string, ISummary, number][] => {
		const cur = summary[`${key}_summary` as keyof IBroadcastSummary] as Record<string, ISummary>
		const totalCount = Object.values(cur).reduce((acc, curr) => acc + curr.count, 0)
		return Object.entries(cur)
			.sort(([, a], [, b]) => b.count - a.count)
			.map(([name, data]) => [name, data, Math.max((data.count / totalCount) * 100, 1)])
	}

	const updateIncPacketHistory = (incPacket: number, incBytes: number) => {
		const now = Date.now()
		setIncPacketHistory(prevHistory => {
			const newHistory = [...prevHistory, { time: date_format(now, 'hh:mm:ss'), packet: incPacket, bytes: incBytes }]
			while (newHistory.length < 30) {
				newHistory.unshift({
					time: date_format(now - (30 - newHistory.length) * 5000, 'hh:mm:ss'),
					packet: 0,
					bytes: 0
				})
			}
			return newHistory.slice(-30)
		})
	}

	const updateDayHistory = (summary: IBroadcastSummary) => {
		const now = new Date()
		const dayHistory = Array.from({ length: 30 }, (_, i) => {
			const date = new Date(now)
			date.setDate(now.getDate() - i)
			const dateStr = date_format(date, 'yyyy-MM-dd')
			const day = summary.day_summary[dateStr]
			return {
				time: dateStr,
				packet: day?.count || 0,
				bytes: day?.size || 0
			}
		}).reverse()
		setDayHistory(dayHistory)
	}

	const parseSummary = (summary: IBroadcastSummary) => {
		const data: ISummaryData = {
			inc_packet: 0,
			inc_bytes: 0,
			total_packet: 0,
			total_bytes: 0,
			day_packet: 0,
			day_bytes: 0,
			country: [],
			city: [],
			eth_type: [],
			ip_proto: [],
			match: [],
			dst_port: [],
			black: [],
			input: []
		}
		for (const key of ['country', 'city', 'eth_type', 'ip_proto', 'match'] as const) {
			data[key] = parseSummaryItem(summary, key)
		}
		data.dst_port = parseSummaryItem(summary, 'dst_port').map(([port, data, percentage]) => [Number(port), data, percentage])
		const [totalPacket, totalBytes] = Object.values(summary.day_summary).reduce(
			(acc, curr) => [acc[0] + curr.count, acc[1] + curr.size],
			[0, 0]
		)
		let isFirst = lastTotalBytesRef.current === 0
		data.total_packet = totalPacket
		data.total_bytes = totalBytes
		data.inc_bytes = totalBytes - lastTotalBytesRef.current
		lastTotalBytesRef.current = totalBytes
		data.inc_packet = totalPacket - lastTotalPacketRef.current
		lastTotalPacketRef.current = totalPacket
		data.input = Object.values(summary.input_packets).sort((a, b) => b.summary.count - a.summary.count)
		if (!isFirst) {
			updateIncPacketHistory(data.inc_packet, data.inc_bytes)
		}
		updateDayHistory(summary)

		const now = new Date()
		const todaySummary = summary.day_summary[date_format(now, 'yyyy-MM-dd')]
		if (todaySummary) {
			data.day_packet = todaySummary.count
			data.day_bytes = todaySummary.size
		}
		setSummaryData(data)
	}

	useEffect(() => {
		if (socket) {
			socket.close()
		}
		let ws: WebSocket | null = null

		const connect = () => {
			ws = new WebSocket(url)
			ws.onopen = () => setIsConnected(true)
			ws.onclose = () => {
				setIsConnected(false)
				if (userClose) {
					return
				}
				setTimeout(connect, 2000)
			}
			ws.onmessage = event => {
				let message: IWebSocketMessage
				try {
					let data = event.data
					if (typeof data !== 'string') {
						data = data + ''
					}
					message = JSON.parse(data)
				} catch {
					return
				}
				if (!message?.action) {
					return
				}
				if (message.action === 'broadcast-summary') {
					parseSummary(message.payload as IBroadcastSummary)
				} else if (message.action === 'broadcast-black') {
					console.log('收到黑名单事件:', message.payload)
				} else if (message.action === 'callback') {
					const callback = callbackMap.current[message.id]
					if (callback) {
						callback.resolve(message.payload)
						delete callbackMap.current[message.id]
					}
				} else if (message.action === 'callback-error') {
					const callback = callbackMap.current[message.id]
					if (callback) {
						callback.reject(new Error(message.payload as string))
						delete callbackMap.current[message.id]
					}
				}
			}
			setSocket(ws)
		}

		connect()

		return () => {
			ws?.close()
		}
	}, [url, userClose])

	const invoke = useCallback(
		<T = unknown>(action: string, payload?: unknown, timeout = 1000 * 30): Promise<T> => {
			if (socket && isConnected) {
				return new Promise((resolve, reject) => {
					const message: IWebSocketMessage = {
						id: Date.now().toString(),
						action,
						payload
					}
					const timer = setTimeout(() => {
						delete callbackMap.current[message.id]
						reject(new Error('请求超时'))
					}, timeout)
					callbackMap.current[message.id] = {
						create_time: Date.now(),
						resolve: (data: unknown) => {
							clearTimeout(timer)
							resolve(data as T)
						},
						reject: (error: Error) => {
							clearTimeout(timer)
							reject(error)
						}
					}
					socket.send(JSON.stringify(message))
				})
			} else {
				throw new Error('WebSocket未连接，无法发送消息')
			}
		},
		[socket, isConnected]
	)

	useEffect(() => {
		if (isConnected) {
			socketAction.getSummary()
		}
	}, [isConnected])

	const socketAction = {
		/** 心跳 */
		ping: async () => (await invoke<string>('ping')) === 'pong',
		/** 获取链路类型 */
		getLinkType: () => invoke<'offload' | 'driver' | 'generic'>('get_link_type'),
		/** 获取统计数据 */
		getSummary: async () => parseSummary(await invoke<IBroadcastSummary>('get_summary')),
		/** 获取规则 */
		getRules: () => invoke<IRule[]>('get_rules'),
		/** 设置规则 */
		setRules: (rules: IRule[]) => invoke<number>('set_rules', rules),
		/** 获取匹配列表 */
		getMatchList: (query: IWebSocketMatchQueryPayload) => invoke<IEnhancedPacketInfo[]>('get_match_list', query),
		/** 获取黑名单 */
		getBlackList: () => invoke<IConfigBlack[]>('get_black_list'),
		/** 修改黑名单 */
		changeBlack: (payload: IWebSocketChangeBlackListPayload) => invoke<boolean>('change_black', payload),
		/** 获取系统资源使用情况 */
		getSystemResourceUsage: () => invoke<ISystemResourceUsage>('get_system_resource_usage'),
		/** 修改广播状态 */
		changeBroadcastStatus: (enable: boolean) => invoke<boolean>('change_broadcast_status', enable)
	}

	return {
		isConnected,
		summaryData,
		incPacketHistory,
		dayHistory,
		userClose,
		setUserClose,
		socketAction
	}
}

export default useWebSocket
export type SocketAction = ReturnType<typeof useWebSocket>['socketAction']
