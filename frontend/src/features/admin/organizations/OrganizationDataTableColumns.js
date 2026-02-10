import { h } from 'vue'
import { RouterLink } from 'vue-router'
import OrganizationDataTableDropdown from '@/features/admin/organizations/OrganizationDataTableDropdown.vue'
import { format } from 'date-fns'

export const columns = (t) => [
  {
    accessorKey: 'name',
    header: () => h('div', { class: 'text-center' }, t('globals.terms.name')),
    cell: ({ row }) =>
      h(
        'div',
        { class: 'text-center' },
        h(
          RouterLink,
          {
            to: { name: 'organization-detail', params: { id: row.original.id } },
            class: 'text-primary hover:underline'
          },
          () => row.getValue('name')
        )
      )
  },
  {
    accessorKey: 'description',
    header: () => h('div', { class: 'text-center' }, 'Description'),
    cell: ({ row }) =>
      h('div', { class: 'text-center truncate max-w-[200px]' }, row.original.description || '—')
  },
  {
    accessorKey: 'updated_at',
    header: () => h('div', { class: 'text-center' }, 'Updated at'),
    cell: ({ row }) =>
      h('div', { class: 'text-center' }, format(new Date(row.getValue('updated_at')), 'PPpp'))
  },
  {
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => {
      const org = row.original
      return h('div', { class: 'relative' }, h(OrganizationDataTableDropdown, { org }))
    }
  }
]
