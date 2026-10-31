import { defaultSchema } from 'rehype-sanitize'

// 在默认 Markdown 清理规则上允许内部维基链接协议、音视频标签及渲染插件生成的属性。
// 属性列表通过新数组合并，避免修改清理库提供的共享默认数组。
export const previewSanitizeSchema = {
  ...defaultSchema,
  clobberPrefix: '',
  protocols: {
    ...defaultSchema.protocols,
    href: [
      ...(defaultSchema.protocols?.href ?? []),
      'wikilink-notfound',
      'wikilink-ambiguous',
    ],
    src: [...(defaultSchema.protocols?.src ?? []), 'http', 'https'],
  },
  tagNames: [...(defaultSchema.tagNames || []), 'audio', 'video', 'mark'],
  attributes: {
    ...defaultSchema.attributes,
    '*': [
      ...(defaultSchema.attributes?.['*'] || []),
      'class',
      'className',
      'data-leafdoc-generated-id',
      'data-line',
      'data-line-numbers',
      'style',
    ],
    code: [
      ...(defaultSchema.attributes?.code || []),
      'className',
      'class',
      'data-line',
      'data-line-numbers',
    ],
    audio: [...(defaultSchema.attributes?.audio || []), 'controls', 'src'],
    video: [
      ...(defaultSchema.attributes?.video || []),
      'controls',
      'src',
      'preload',
    ],
  },
}
