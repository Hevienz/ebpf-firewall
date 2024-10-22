export function classNames(...classes: unknown[]): string {
	return classes.filter(Boolean).join(' ')
}

/**
 * 日期格式化
 * @param dt 日期
 * @param fmt 格式 默认 yyyy-MM-dd hh:mm:ss
 * @returns 格式化后的日期
 */
export function date_format(dt: Date | number | string | undefined, fmt = 'yyyy-MM-dd hh:mm:ss') {
	const date = dt ? new Date(dt) : new Date()
	const o: Record<string, number> = {
		'M+': date.getMonth() + 1, // 月份
		'd+': date.getDate(), // 日
		'h+': date.getHours(), // 小时
		'm+': date.getMinutes(), // 分
		's+': date.getSeconds(), // 秒
		'q+': Math.floor((date.getMonth() + 3) / 3), // 季度
		S: date.getMilliseconds() // 毫秒
	}
	fmt = fmt.replace(/(y+)/, match => (date.getFullYear() + '').slice(-match.length))
	for (const [k, v] of Object.entries(o)) {
		fmt = fmt.replace(new RegExp(`(${k})`), match => (match.length === 1 ? String(v) : ('00' + v).slice(-match.length)))
	}
	return fmt
}

export function thousandBitSeparator(num: number): string {
	if (!num) return '0'
	return num.toString().replace(/(\d)(?=(?:\d{3})+$)/g, '$1,')
}

export function formatByteSizeToStr(val: number, unit = 1000, fractionDigits = 2): string {
	const { n, unit: u } = formatByteSize(val, unit, fractionDigits)
	return `${n} ${u}`
}

export function formatByteSize(val: number, unit = 1000, fractionDigits = 2): { unit: string; n: string } {
	if (!val) {
		val = 0
	}
	if (val > unit * unit * unit) {
		return {
			n: (val / (unit * unit * unit)).toFixed(fractionDigits),
			unit: 'GB'
		}
	} else if (val > unit * unit) {
		return { n: (val / (unit * unit)).toFixed(fractionDigits), unit: 'MB' }
	} else if (val > unit) {
		return { n: (val / unit).toFixed(fractionDigits), unit: 'KB' }
	} else {
		return { n: val + '', unit: 'Bit' }
	}
}

export function getDateDiff(dateTimeStamp: any): string {
	if (typeof dateTimeStamp === 'string') {
		dateTimeStamp = parseInt(dateTimeStamp)
	} else if (dateTimeStamp instanceof Date) {
		dateTimeStamp = dateTimeStamp.getTime()
	}
	const m = 1000 * 10
	const minute = 1000 * 60
	const hour = minute * 60
	const diffValue = Date.now() - dateTimeStamp
	const hourC = diffValue / hour
	const minC = diffValue / minute
	const mC = diffValue / m
	let result = '刚刚'
	if (hourC >= 1) {
		const l = new Date(dateTimeStamp).setHours(0, 0, 0, 0)
		const r = new Date().setHours(0, 0, 0, 0)
		if (l - r === 0) {
			result = '今天 ' + date_format(dateTimeStamp, 'hh:mm:ss')
		} else if (l - r === -86400000) {
			result = '昨天 ' + date_format(dateTimeStamp, 'hh:mm:ss')
		} else if (l - r === -86400000 * 2) {
			result = '前天 ' + date_format(dateTimeStamp, 'hh:mm:ss')
		} else {
			result = date_format(dateTimeStamp, 'MM-dd hh:mm:ss')
		}
	} else if (minC >= 1) {
		result = parseInt(minC + '') + '分钟 之前'
	} else if (mC >= 1) {
		result = parseInt(mC * 10 + '') + '秒 之前'
	}
	return result
}
