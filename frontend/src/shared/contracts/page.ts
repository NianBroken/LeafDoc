// Page kinds are shared by tree, search and import response contracts.
export const NODE_KIND_PAGE = 'page'
export const NODE_KIND_SECTION = 'section'
export type PageKind = typeof NODE_KIND_PAGE | typeof NODE_KIND_SECTION
