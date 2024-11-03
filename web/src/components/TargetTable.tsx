import { memo, useEffect, useState } from 'react'
import { TargetStatistic } from 'types'
import * as api from '../api'
import { Button, PaginationProps, Table } from '@arco-design/web-react'
import { SorterInfo } from '@arco-design/web-react/es/Table/interface'
import AnimatedCounter from './AnimatedCounter'
import { formatByteSizeToStr, getDateDiff, thousandBitSeparator } from 'utils'
import { getEthernetTypeString, getIPProtocolString } from 'enums'

export const TargetTable: React.FC<{ skey: string }> = memo(({ skey }) => {
	const [targets, setTargets] = useState<TargetStatistic[]>([])
	const [pagination, setPagination] = useState({
		sizeCanChange: true,
		showTotal: true,
		total: 0,
		pageSize: 20,
		current: 1,
		pageSizeChangeResetCurrent: true
	})
	const [sorting, setSorting] = useState<{
		field: string
		order: 'asc' | 'desc'
	}>({
		field: 'last_seen_at',
		order: 'desc'
	})

	const fetchData = async () => {
		if (skey === '') return
		const res = await api.getTargets(skey, pagination.current, pagination.pageSize, sorting.field, sorting.order)
		setTargets(res.items)
		setPagination(prev => ({
			...prev,
			total: res.total
		}))
	}

	const onChangeTable = (pagination: PaginationProps, sorter: SorterInfo | SorterInfo[]) => {
		const { current, pageSize } = pagination
		setPagination(prev => ({ ...prev, current: current ?? 1, pageSize: pageSize ?? 20 }))
		if (!Array.isArray(sorter)) {
			sorter = [sorter]
		}
		if (sorter.length > 0) {
			const sort = sorter[0]
			console.log(sort)
			if (sort.direction) {
				setSorting({ field: sort.field as string, order: sort.direction === 'ascend' ? 'asc' : 'desc' })
				return
			}
		}
		console.log('reset')
		setSorting(prev => ({ ...prev, field: 'last_seen_at', order: 'desc' }))
	}

	useEffect(() => {
		fetchData()
	}, [skey, pagination.current, pagination.pageSize, sorting.field, sorting.order])

	return (
		<div className="w-full overflow-hidden">
			<div className="mb-2 flex justify-end">
				<Button onClick={() => fetchData()}>刷新</Button>
			</div>
			<Table
				data={targets}
				pagination={pagination}
				onChange={onChangeTable}
				stripe={true}
				size="small"
				scroll={{ y: 400 }}
				rowKey={(record: any) => `${record.target_mac}-${record.target_ip}-${record.target_port}`}
				columns={[
					{
						title: '目标MAC',
						dataIndex: 'target_mac',
						ellipsis: true,
						width: 150,
						tooltip: true,
						align: 'center',
						render: value => value || '-'
					},
					{
						title: '目标IP',
						dataIndex: 'target_ip',
						ellipsis: true,
						width: 200,
						tooltip: true,
						align: 'center',
						render: value => value || '-'
					},
					{
						title: '目标端口',
						dataIndex: 'target_port',
						ellipsis: true,
						width: 100,
						tooltip: true,
						align: 'center',
						render: value => value || '-'
					},
					{
						title: 'Eth类型',
						dataIndex: 'eth_type',
						width: 100,
						tooltip: true,
						align: 'center',
						sorter: true,
						render: value => getEthernetTypeString(value)
					},
					{
						title: 'IP协议',
						dataIndex: 'ip_proto',
						width: 100,
						tooltip: true,
						align: 'center',
						sorter: true,
						render: value => getIPProtocolString(value)
					},
					{
						title: '包数',
						dataIndex: 'total_packets',
						width: 80,
						tooltip: true,
						align: 'center',
						sorter: true,
						render: value => (
							<AnimatedCounter endValue={value < 1 ? 0 : value} duration={500} formatter={thousandBitSeparator} />
						)
					},
					{
						title: '字节数',
						dataIndex: 'total_bytes',
						width: 100,
						tooltip: true,
						align: 'center',
						sorter: true,
						render: value => <AnimatedCounter endValue={value < 1 ? 0 : value} duration={500} formatter={formatByteSizeToStr} />
					},
					{
						title: '上次访问',
						dataIndex: 'last_seen_at',
						width: 120,
						tooltip: true,
						align: 'center',
						sorter: true,
						defaultSortOrder: 'descend',
						render: value => getDateDiff(value * 1000)
					}
				]}
			/>
		</div>
	)
})
