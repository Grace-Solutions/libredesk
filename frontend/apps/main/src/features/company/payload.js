// The company picker holds ids as strings so the combobox can compare them, but the
// API takes parent_id as a number or null. Convert on the way out.
export const toCompanyPayload = (values) => ({
    ...values,
    parent_id: toCompanyID(values.parent_id)
})

export const toCompanyID = (value) => {
    const id = Number(value)
    return Number.isInteger(id) && id > 0 ? id : null
}
