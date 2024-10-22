import React from 'react'
import { IconArrowUp, IconInfoCircle } from '@arco-design/web-react/icon'
import ReactECharts from 'echarts-for-react'
import { thousandBitSeparator, formatByteSizeToStr } from 'utils'
import { Alert } from '@arco-design/web-react'
import { IPacketData, ISummaryData } from '../types'
import AnimatedCounter from './AnimatedCounter'
import { PacketTable } from './PacketTable'

const Dashboard: React.FC<{
	summaryData: ISummaryData
	incPacketHistory: IPacketData[]
	dayHistory: IPacketData[]
}> = ({ summaryData, incPacketHistory, dayHistory }) => {
	function buildOption(data: IPacketData[], tooltipFormatter: (params: any) => string) {
		return {
			xAxis: {
				type: 'category',
				data: data.map(x => x.time),
				axisLabel: { show: false },
				axisLine: { show: false },
				axisTick: { show: false }
			},
			yAxis: [
				{
					type: 'value',
					name: '包数',
					axisLabel: { show: false },
					axisLine: { show: false },
					axisTick: { show: false },
					splitLine: { show: false }
				},
				{
					type: 'value',
					name: '字节数',
					axisLabel: { show: false },
					axisLine: { show: false },
					axisTick: { show: false },
					splitLine: { show: false }
				}
			],
			series: [
				{
					data: data.map(x => x.packet),
					type: 'bar',
					itemStyle: {
						color: '#10B981'
					},
					yAxisIndex: 1,
					barWidth: '60%',
					showBackground: true,
					backgroundStyle: {
						color: 'rgba(180, 180, 180, 0.2)'
					}
				},
				{
					data: data.map(x => x.bytes),
					type: 'line',
					smooth: true,
					itemStyle: {
						color: '#34D399' // 添加这个颜色属性
					},
					lineStyle: {
						color: '#34D399' // 添加这个颜色属性
					},
					areaStyle: {
						color: 'rgba(52, 211, 153, 0.2)' // 添加这个区域填充颜色
					}
				}
			],
			tooltip: {
				trigger: 'axis',
				axisPointer: {
					type: 'shadow'
				},
				formatter: (params: any) => {
					return tooltipFormatter(params)
				}
			},
			grid: {
				left: '0',
				right: '0',
				bottom: '3%',
				top: '3%',
				containLabel: true
			}
		}
	}
	const incPacketOption = buildOption(
		incPacketHistory,
		params =>
			`${params[0].axisValue} 入站 ${thousandBitSeparator(params[0].data)} 个数据包，${formatByteSizeToStr(params[1].data)} 字节`
	)

	const thirtyDaysVisitOption = buildOption(
		dayHistory,
		params =>
			`${params[0].axisValue} 入站 ${thousandBitSeparator(params[0].data)} 个数据包，${formatByteSizeToStr(params[1].data)} 字节`
	)

	const renderSummary = (label: string, key: string) => {
		return (
			<div key={key} className="bg-white rounded-lg shadow-sm">
				<h3 className="text-lg px-4 py-2.5 border-b border-gray-200">{label}</h3>
				<div className="space-y-2 h-[250px] overflow-y-auto">
					{(summaryData as any)[key]?.length ? (
						(summaryData as any)[key].map(([name, data, percentage]: any, index: number) => (
							<div
								key={`${key}-${name}-${index}`}
								className="flex flex-col text-sm p-3 rounded-md transition duration-300 ease-in-out hover:bg-gray-100"
							>
								<div className="flex justify-between items-center mb-2">
									<span className="truncate flex-1">{name === '-' || !name ? '未知' : name}</span>
									<span className="ml-2">
										<AnimatedCounter endValue={data?.count || 0} duration={1000} formatter={thousandBitSeparator} />
									</span>
								</div>
								<div className="relative h-2 bg-gray-200 rounded-full overflow-hidden">
									<div
										className="absolute top-0 left-0 h-full bg-teal-500 transition-all duration-300 ease-in-out"
										style={{
											width: `${percentage || 0}%`
										}}
									></div>
								</div>
							</div>
						))
					) : (
						<div className="flex justify-center items-center h-full text-gray-400 text-sm">暂无数据</div>
					)}
				</div>
			</div>
		)
	}

	return (
		<div className="space-y-4">
			<Alert
				type="info"
				icon={<IconInfoCircle />}
				content={
					<span>
						当前程序基于 XDP 实现，当前仅计算 <strong>入站流量</strong> 数据。
					</span>
				}
			/>
			<div className="grid grid-cols-12 gap-4">
				<div className="col-span-3 flex justify-between items-center bg-white p-4 rounded-lg shadow-sm">
					<div>
						<div className="flex mb-2">
							<h3 className="text-sm text-gray-500">总包数</h3>
						</div>
						<p className="text-2xl font-semibold">
							<AnimatedCounter endValue={summaryData.total_packet} duration={1000} formatter={thousandBitSeparator} />
						</p>
					</div>
					<div>
						<div className="flex mb-2">
							<h3 className="text-sm text-gray-500">今日总包数</h3>
						</div>
						<p className="text-2xl font-semibold">
							<AnimatedCounter endValue={summaryData.day_packet} duration={1000} formatter={thousandBitSeparator} />
							<IconArrowUp className="text-green-500 w-3 h-3 ml-2" />
							<span className="text-green-500 text-sm">
								<AnimatedCounter endValue={summaryData.inc_packet} duration={1000} formatter={thousandBitSeparator} />
							</span>
						</p>
					</div>
				</div>
				<div className="col-span-3 flex justify-between items-center bg-white p-4 rounded-lg shadow-sm">
					<div>
						<div className="flex mb-2">
							<h3 className="text-sm text-gray-500">总字节数</h3>
						</div>
						<p className="text-2xl font-semibold">
							<AnimatedCounter endValue={summaryData.total_bytes} duration={1000} formatter={formatByteSizeToStr} />
						</p>
					</div>
					<div>
						<div className="flex mb-2">
							<h3 className="text-sm text-gray-500">今日总字节数</h3>
						</div>
						<p className="text-2xl font-semibold">
							<AnimatedCounter endValue={summaryData.day_bytes} duration={1000} formatter={formatByteSizeToStr} />
							<IconArrowUp className="text-green-500 w-3 h-3 ml-2" />
							<span className="text-green-500 text-sm">
								<AnimatedCounter endValue={summaryData.inc_bytes} duration={1000} formatter={formatByteSizeToStr} />
							</span>
						</p>
					</div>
				</div>
				<div className="col-span-6">
					<div className="bg-white p-4 rounded-lg shadow-sm">
						<ReactECharts option={incPacketOption} style={{ height: '60px', width: '100%' }} />
					</div>
				</div>
			</div>

			<div className="grid grid-cols-2 gap-4">
				<div>
					<div className="grid grid-cols-3 gap-4 mb-4">
						{[
							{ label: '访问国家', key: 'country' },
							{ label: '访问城市', key: 'city' },
							{ label: '访问端口', key: 'dst_port' }
						].map(x => renderSummary(x.label, x.key))}
					</div>
					<div className="grid grid-cols-3 gap-4">
						{[
							{ label: 'ETH 类型', key: 'eth_type' },
							{ label: 'IP 协议', key: 'ip_proto' },
							{ label: '命中规则', key: 'match' }
						].map(x => renderSummary(x.label, x.key))}
					</div>
				</div>
				<div>
					<div className="bg-white rounded-lg shadow-sm">
						<h3 className="text-lg px-4 py-2.5 border-b border-gray-200">来源数据</h3>
						<div className="space-y-2 h-[565px] p-4 overflow-y-auto">
							<PacketTable packets={summaryData.input} />
						</div>
					</div>
				</div>
			</div>

			<div className="bg-white p-4 rounded-lg shadow-sm">
				<h3 className="text-lg mb-4">30 天访问情况</h3>
				<ReactECharts option={thirtyDaysVisitOption} style={{ height: '200px', width: '100%' }} />
			</div>
		</div>
	)
}

export default Dashboard
