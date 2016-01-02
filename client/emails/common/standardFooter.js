import React from 'react'

import { A, Span, textDefaults } from 'react-html-email'
import Footer from './Footer'


export default (
  <Footer>
    <Span {...textDefaults} fontSize={13} color="#7d7d7d">
      This message was sent to <A {...textDefaults} textDecoration="none" href="mailto:{{.AccountEmailAddress}}">{'{{.AccountEmailAddress}}'}</A> because an account is registered on <A {...textDefaults} textDecoration="none" href="{{.SiteURL}}">{'{{.SiteURLShort}}'}</A> with this email address.
    </Span>
  </Footer>
)
