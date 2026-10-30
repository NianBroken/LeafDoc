import { Button } from '@/shared/components/ui/button'
import { NODE_KIND_PAGE, NODE_KIND_SECTION, Page } from '@/shared/lib/api/pages'
import i18next from '@/shared/lib/i18n'
import { DIALOG_ADD_PAGE } from '@/shared/lib/registries/instances'
import { useIsReadOnly } from '@/shared/lib/useIsReadOnly'
import { useDialogsStore } from '@/shared/state/dialogs'
import { useTreeStore } from '@/features/tree/state/tree'
import { FilePlus, FolderPlus } from 'lucide-react'

const t = (key: string, opts?: Record<string, unknown>) =>
  i18next.t(key, { ...opts, ns: 'viewer' })

type EmptySectionChildrenListProps = {
  page: Page
}

export default function EmptySectionChildrenList({
  page,
}: EmptySectionChildrenListProps) {
  const getPageById = useTreeStore((s) => s.getPageById)
  const node = getPageById(page.id)
  const openDialog = useDialogsStore((s) => s.openDialog)
  const tree = useTreeStore((s) => s.tree)
  const isReadOnly = useIsReadOnly()

  if (!tree) {
    return null
  }

  if (page.kind !== NODE_KIND_SECTION) {
    return null
  }

  // Authored section content is displayed by the normal Markdown viewer.
  if (page.content && page.content.trim().length > 0) {
    return null
  }

  if (!node) {
    return null
  }

  return (
    <nav
      aria-label={t('section.subpagesAriaLabel', { title: page.title })}
      className="child-list__section"
    >
      <div>
        <h2 className="child-list__section-title mb-1 grow">
          {t('section.emptyTitle')}
        </h2>
        <p className="text-muted text-sm">
          {isReadOnly
            ? t('section.emptyDescriptionReadOnly', { title: page.title })
            : t('section.emptyDescriptionEditor', { title: page.title })}
        </p>
      </div>
      {!isReadOnly && (
        <div className="mb-2 flex w-full justify-end">
          <Button
            onClick={() =>
              openDialog(DIALOG_ADD_PAGE, {
                parentId: page.id,
                nodeKind: NODE_KIND_PAGE,
              })
            }
            variant="default"
            size="sm"
          >
            <FilePlus size={16} />
            {t('section.addPage')}
          </Button>
          <Button
            onClick={() =>
              openDialog(DIALOG_ADD_PAGE, {
                parentId: page.id,
                nodeKind: NODE_KIND_SECTION,
              })
            }
            variant="default"
            size="sm"
            className="ml-2"
          >
            <FolderPlus size={16} />
            {t('section.addSection')}
          </Button>
        </div>
      )}
    </nav>
  )
}
