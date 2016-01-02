import React from 'react'

import { Item, Span, A } from 'react-html-email'
import { StandardEmail, TopBubbleBox, BodyBox, standardFooter, textDefaults } from './common'


module.exports = (
  <StandardEmail>
    <TopBubbleBox logo="logo-warning.png" padding={15}>
      <Item align="center">
        <Span {...textDefaults} fontSize={20}>Your password has been changed.</Span>
      </Item>
    </TopBubbleBox>
    <BodyBox>
      <Item>
        <Span {...textDefaults}>Hey, just keeping you in the loop. If you just updated your <A {...textDefaults} href="{{.SiteURL}}">{'{{.SiteName}}'}</A> password, you're good to go!</Span>
      </Item>
      <Item>
        <Span {...textDefaults}>If you did not change your password and suspect something fishy is going on, please reply to this email immediately.</Span>
      </Item>
    </BodyBox>
    {standardFooter}
  </StandardEmail>
)
