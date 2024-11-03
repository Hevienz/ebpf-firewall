import { useState, useEffect, useRef } from 'react'
import { date_format } from '../utils'
import { MetricsReport, DimensionKey, Statistics, IncrementMetrics, DeltaMetrics } from '../types'
import * as api from '../api'

function useMetrics(pollInterval: number) {
	const [metricsReport, setMetricsReport] = useState<MetricsReport>({
		total_packets: 0,
		total_bytes: 0,
		day: [],
		dimension: {} as Record<DimensionKey, Statistics[]>
	})
	const [todayData, setTodayData] = useState<DeltaMetrics>({
		packets: 0,
		bytes: 0
	})
	const [trafficDeltas, setTrafficDeltas] = useState<IncrementMetrics[]>([])
	const lastDelta = useRef({
		packets: 0,
		bytes: 0
	})
	const [deltaMetrics, setDeltaMetrics] = useState<DeltaMetrics>({
		packets: 0,
		bytes: 0
	})
	const first = useRef(true)
	const pollingRef = useRef<ReturnType<typeof setInterval>>()

	const calculateTrafficDelta = (report: MetricsReport, time: string) => {
		let deltaPackets = 0
		let deltaBytes = 0
		if (first.current) {
			lastDelta.current.packets = report.total_packets
			lastDelta.current.bytes = report.total_bytes
			first.current = false
		} else {
			deltaPackets = report.total_packets - lastDelta.current.packets
			deltaBytes = report.total_bytes - lastDelta.current.bytes
			setDeltaMetrics({
				packets: deltaPackets,
				bytes: deltaBytes
			})
			lastDelta.current.packets = report.total_packets
			lastDelta.current.bytes = report.total_bytes
		}
		const now = new Date()
		const today = report.day.find(x => x.key === time)
		if (today) {
			setTodayData({
				packets: today.total_packets,
				bytes: today.total_bytes
			})
		}
		setTrafficDeltas(prev => {
			const newHistory = [
				...prev,
				{
					time: date_format(now, 'hh:mm:ss'),
					packets: deltaPackets,
					bytes: deltaBytes
				}
			]
			while (newHistory.length < 30) {
				newHistory.unshift({
					time: date_format(now.getTime() - (30 - newHistory.length) * 5000, 'hh:mm:ss'),
					packets: 0,
					bytes: 0
				})
			}
			return newHistory.slice(-30)
		})

		report.day = Array.from({ length: 30 }, (_, i) => {
			const date = new Date(now)
			date.setDate(now.getDate() - i)
			const day = report.day.find(x => x.key === date_format(date, 'yyyyMMdd'))
			return {
				key: date_format(date, 'yyyy-MM-dd'),
				total_packets: day?.total_packets || 0,
				total_bytes: day?.total_bytes || 0
			}
		}).reverse()
	}

	useEffect(() => {
		if (pollInterval < 1) {
			return
		}
		const fetchData = async () => {
			try {
				const report = await api.getMetricsReport()
				const now = new Date()
				const time = date_format(now, 'yyyyMMdd')
				const day = report.day.find(x => x.key === time)
				calculateTrafficDelta(report, time)

				setMetricsReport(report)
			} catch (error) {
				console.error(error)
			}
		}

		fetchData()
		pollingRef.current = setInterval(fetchData, pollInterval * 1000)
		return () => {
			if (pollingRef.current) {
				clearInterval(pollingRef.current)
			}
		}
	}, [pollInterval])

	return {
		report: metricsReport,
		todayData,
		trafficDeltas,
		deltaMetrics
	}
}

export default useMetrics
