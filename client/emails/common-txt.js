const standardFooter = `
this message was sent to {{.AccountEmailAddress}} because an account is registered on {{.SiteURL}} with this email address.

to change your email notification preferences, click here:

{{.EmailPreferencesURL}}
`.trim()

export default { standardFooter }
