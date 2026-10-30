import { DIALOG_EDIT_PAGE_METADATA } from '@/shared/lib/registries/instances'
import { getParentWikiRoutePath, toWikiLookupPath } from '@/shared/lib/wikiPath'
import { useAppMode } from '@/shared/lib/useAppMode'
import { useIsMobile } from '@/shared/lib/useIsMobile'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useTreeStore } from '@/features/tree/state/tree'
import { Pencil } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { TooltipWrapper } from '@/shared/components/TooltipWrapper'
import { usePageEditorStore } from '@/features/editor/pageEditorStore'
import { isDirtyState } from '@/features/editor/pageEditorStore'

export function EditorTitleBar() {
  const { t } = useTranslation('editor')
  const isMobile = useIsMobile()
  const appMode = useAppMode()
  const page = usePageEditorStore((state) => state.page)
  const title = usePageEditorStore((state) => state.title)
  const slug = usePageEditorStore((state) => state.slug)
  const setTitle = usePageEditorStore((state) => state.setTitle)
  const setSlug = usePageEditorStore((state) => state.setSlug)
  const openDialog = useDialogsStore((s) => s.openDialog)
  const getPageByPath = useTreeStore((state) => state.getPageByPath)
  const dirty = usePageEditorStore(isDirtyState)

  const onEditClicked = () => {
    if (!page) return

    const parentId = () => {
      const parentPath = toWikiLookupPath(getParentWikiRoutePath(page.path))
      const p = getPageByPath(parentPath)
      if (!p) return ''
      return p.id
    }

    openDialog(DIALOG_EDIT_PAGE_METADATA, {
      title: title,
      currentId: page.id,
      itemKind: page.kind,
      slug: slug,
      parentId: parentId(),
      onChange: (title: string, slug: string) => {
        setTitle(title)
        setSlug(slug)
      },
    })
  }

  if (appMode !== 'edit') {
    return null
  }

  if (page == null) {
    return null
  }

  return (
    <div className="editor-title-bar">
      <button
        onClick={onEditClicked}
        className="editor-title-bar__button"
        data-testid="edit-page-metadata-button"
      >
        <TooltipWrapper
          label={title}
          side="top"
          align="start"
          parentClassName="min-w-0 max-w-full items-center gap-2"
        >
          {title && <span className="editor-title-bar__title">{title}</span>}
          <Pencil size={16} className="editor-title-bar__icon" />
          {dirty && !isMobile && (
            <span className="editor-title-bar__dirty-indicator">
              {t('titleBar.dirtyIndicator')}
            </span>
          )}

          {dirty && isMobile && (
            <span className="editor-title-bar__dirty-indicator">*</span>
          )}
        </TooltipWrapper>
      </button>
      <span className="editor-title-bar__slug">{slug}</span>
    </div>
  )
}
