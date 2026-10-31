import { slugifyHeadline } from '@/features/preview/rehypeLineNumber'

export type TocEntry = {
  // 目录只输出一至三级标题，文本为简化后的行内 Markdown，id 与页面标题锚点规则一致。
  level: 1 | 2 | 3
  text: string
  id: string
}

function stripInlineMarkdown(text: string): string {
  // 移除注释、HTML 标签及常见链接和行内代码标记，保留读者看到的文字部分。
  return text
    .replace(/<!--[\s\S]*?-->/g, '')
    .replace(/<[^>]*>/g, '')
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/\[([^\]]*)\]\[[^\]]*\]/g, '$1')
    .replace(/\[([^\]]*)\](?!\(|\[)/g, '$1')
    .replace(/`([^`]+)`/g, '$1')
    .replace(/[*_]{1,3}([^*_]+)[*_]{1,3}/g, '$1')
    .trim()
}

export function extractTocEntries(markdown: string): TocEntry[] {
  // 按行扫描 Markdown，追踪围栏代码范围并对重复标题累计生成稳定后缀。
  const lines = markdown.split('\n')
  const entries: TocEntry[] = []
  const slugCounts: Record<string, number> = {}
  let inCodeBlock = false
  let fenceChar: string | null = null
  let fenceLength = 0

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]

    if (inCodeBlock) {
      // 只有与开围栏符号相同且长度不短于开围栏的独立标记才能结束代码区域。
      const closeMatch = line.match(/^ {0,3}(`{3,}|~{3,})[ \t]*$/)
      if (
        closeMatch &&
        closeMatch[1][0] === fenceChar &&
        closeMatch[1].length >= fenceLength
      ) {
        inCodeBlock = false
        fenceChar = null
        fenceLength = 0
      }
      continue
    }

    const openMatch = line.match(/^ {0,3}(`{3,}|~{3,})(.*)$/)
    if (openMatch) {
      inCodeBlock = true
      fenceChar = openMatch[1][0]
      fenceLength = openMatch[1].length
      continue
    }

    // 六级 ATX 标题均参与重复 ID 计数，只有前三级进入目录列表。
    const atxMatch = line.match(/^(#{1,6})\s+(.+?)(?:\s+#+\s*)?$/)
    if (atxMatch) {
      const level = atxMatch[1].length
      const text = stripInlineMarkdown(atxMatch[2])
      if (!text) continue
      const baseSlug = slugifyHeadline(text)
      if (!baseSlug) continue
      const count = slugCounts[baseSlug] ?? 0
      slugCounts[baseSlug] = count + 1
      const id = count === 0 ? baseSlug : `${baseSlug}-${count}`
      if (level <= 3) {
        entries.push({ level: level as 1 | 2 | 3, text, id })
      }
      continue
    }

    // Setext 一级标题由下一行等号标记，内容行和标记行一并消费。
    const nextLine = lines[i + 1]
    if (nextLine !== undefined) {
      const rawText = line.trim()
      if (rawText && /^=+\s*$/.test(nextLine)) {
        const text = stripInlineMarkdown(rawText)
        if (text) {
          const baseSlug = slugifyHeadline(text)
          if (baseSlug) {
            const count = slugCounts[baseSlug] ?? 0
            slugCounts[baseSlug] = count + 1
            entries.push({
              level: 1,
              text,
              id: count === 0 ? baseSlug : `${baseSlug}-${count}`,
            })
          }
        }
        i++
        continue
      }
      // Setext 二级标题由下一行连字符标记。
      if (rawText && /^-+\s*$/.test(nextLine)) {
        const text = stripInlineMarkdown(rawText)
        if (text) {
          const baseSlug = slugifyHeadline(text)
          if (baseSlug) {
            const count = slugCounts[baseSlug] ?? 0
            slugCounts[baseSlug] = count + 1
            entries.push({
              level: 2,
              text,
              id: count === 0 ? baseSlug : `${baseSlug}-${count}`,
            })
          }
        }
        i++
      }
    }
  }

  return entries
}
