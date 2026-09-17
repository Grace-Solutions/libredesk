import * as z from 'zod'
import { phoneNumberSchema } from '@shared-ui/utils/phone.js'

export const createFormSchema = (t) =>
    z.object({
        name: z
            .string({
                required_error: t('globals.messages.required')
            })
            .min(2, {
                message: t('validation.minmax', {
                    min: 2,
                    max: 140
                })
            })
            .max(140, {
                message: t('validation.minmax', {
                    min: 2,
                    max: 140
                })
            }),
        description: z.string().max(300).optional().nullable(),
        website: z.string().max(300).optional().nullable(),
        phone_number: phoneNumberSchema(t).optional().nullable(),
        phone_number_country_code: z.string().optional().nullable(),
        country: z.string().optional().nullable(),
        parent_id: z.union([z.string(), z.number()]).optional().nullable()
    })
