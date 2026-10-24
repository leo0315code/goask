// 相对时间显示
export function timeAgo(iso) {
  const d = new Date(iso)
  if (isNaN(d)) return ''
  const s = (Date.now() - d.getTime()) / 1000
  if (s < 60) return '刚刚'
  if (s < 3600) return Math.floor(s / 60) + ' 分钟前'
  if (s < 86400) return Math.floor(s / 3600) + ' 小时前'
  if (s < 86400 * 30) return Math.floor(s / 86400) + ' 天前'
  return d.toLocaleDateString('zh-CN')
}

// 正文摘要
export function excerpt(text, n = 120) {
  const t = String(text || '').replace(/\s+/g, ' ').trim()
  return t.length > n ? t.slice(0, n) + '…' : t
}

// 逗号分隔的标签输入 → 数组
export function parseTags(input) {
  return String(input || '')
    .split(/[,，\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
}
