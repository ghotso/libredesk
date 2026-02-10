import * as z from 'zod'

export const createPortalFormSchema = (t) =>
  z
    .object({
      portal_enabled: z.boolean().optional().default(false),
      portal_default_inbox_id: z.coerce.number().optional().default(0)
    })
    .refine((data) => !data.portal_enabled || data.portal_default_inbox_id > 0, {
      message: t('admin.general.portalDefaultInboxRequired'),
      path: ['portal_default_inbox_id']
    })
