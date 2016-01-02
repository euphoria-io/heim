import React from 'react'

import { Item, Span, A } from 'react-html-email'
import { StandardEmail, TopBubbleBox, BodyBox, BigButton, standardFooter, textDefaults } from './common'


module.exports = (
  <StandardEmail>
    <TopBubbleBox logo="logo-warning.png" padding={15}>
      <Item align="center">
        <Span {...textDefaults} fontSize={20}>Would you like to reset your password?</Span>
      </Item>
    </TopBubbleBox>
    <BodyBox>
      <Item>
        <Span {...textDefaults}>Hey, we've received a password reset request for your <A {...textDefaults} href="{{.SiteURL}}">{'{{.SiteName}}'}</A> account:</Span>
      </Item>
      <BigButton color="#dca955" href="{{.ResetPasswordURL}}">
        reset your password
      </BigButton>
      <Item>
        <Span {...textDefaults}>If you did not make this request and suspect something fishy is going on, please reply to this email immediately.</Span>
      </Item>
    </BodyBox>
    {standardFooter}
  </StandardEmail>
)
