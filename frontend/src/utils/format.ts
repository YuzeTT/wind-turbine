/** 格式化 ISO 时间字符串为 "YYYY-MM-DD HH:mm:ss" */
export function formatDateTime(iso: string | undefined | null): string {
  if (!iso) return ''
  return iso.substring(0, 19).replace('T', ' ')
}

/** 格式化 ISO 时间字符串为 "YYYY-MM-DD" */
export function formatDate(iso: string | undefined | null): string {
  if (!iso) return ''
  return iso.substring(0, 10)
}
