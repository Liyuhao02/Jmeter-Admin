const SHANGHAI_TZ = 'Asia/Shanghai'

const SERVER_DATETIME_RE = /^(\d{4})-(\d{2})-(\d{2})(?:[ T](\d{2}):(\d{2})(?::(\d{2}))?)?/

export const parseServerDateTime = (input) => {
  if (!input) return null
  if (input instanceof Date) return Number.isNaN(input.getTime()) ? null : input
  if (typeof input === 'number') {
    const d = new Date(input)
    return Number.isNaN(d.getTime()) ? null : d
  }

  const value = String(input).trim()
  if (!value) return null

  const match = value.match(SERVER_DATETIME_RE)
  if (match) {
    const [, y, m, d, hh = '00', mm = '00', ss = '00'] = match
    return new Date(
      Number(y),
      Number(m) - 1,
      Number(d),
      Number(hh),
      Number(mm),
      Number(ss)
    )
  }

  const fallback = new Date(value)
  return Number.isNaN(fallback.getTime()) ? null : fallback
}

export const formatDateTimeInShanghai = (input, options = {}) => {
  const value = String(input || '').trim()
  const match = value.match(SERVER_DATETIME_RE)
  if (match) {
    const [, y, m, d, hh = '00', mm = '00', ss = '00'] = match
    return options.withSeconds
      ? `${y}-${m}-${d} ${hh}:${mm}:${ss}`
      : `${y}-${m}-${d} ${hh}:${mm}`
  }

  const date = parseServerDateTime(value)
  if (!date) return '-'

  const formatter = new Intl.DateTimeFormat('zh-CN', {
    timeZone: SHANGHAI_TZ,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: options.withSeconds ? '2-digit' : undefined,
    hour12: false
  })

  return formatter.format(date).replace(/\//g, '-')
}

export const formatRelativeTimeInShanghai = (input) => {
  const date = parseServerDateTime(input)
  if (!date) return '从未检测'

  const diff = Math.floor((Date.now() - date.getTime()) / 1000)
  if (diff < 0) return '刚刚'
  if (diff < 60) return `${diff}秒前`
  if (diff < 3600) return `${Math.floor(diff / 60)}分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)}小时前`
  return `${Math.floor(diff / 86400)}天前`
}
