import React from 'react'

import { Item, Span, A } from 'react-html-email'
import { StandardEmail, TopBubbleBox, BigButton, BodyBox, Footer, textDefaults } from './common'


module.exports = (
  <StandardEmail>
    <TopBubbleBox logo="logo.png">
      <Item align="center">
        <Span {...textDefaults} fontSize={52}>Hi!</Span>
      </Item>
      <Item align="center">
        <Span {...textDefaults} fontSize={18} color="#9f9f9f">Welcome to {'{{.SiteName}}'} :)</Span>
      </Item>
    </TopBubbleBox>
    <BodyBox>
      <Item align="center">
        <Span {...textDefaults}>Your account is almost ready:</Span>
      </Item>
      <BigButton color="#80c080" href="{{.VerifyEmailURL}}">
        verify your email address
      </BigButton>
      <Item>
        <Span {...textDefaults}>We hope you have a wonderful time on <A {...textDefaults} href="{{.SiteURL}}">{'{{.SiteName}}'}</A>. If you have any questions or comments, feel free to <A {...textDefaults} href="mailto:{{.HelpAddress}}">contact us</A>.</Span>
      </Item>
    </BodyBox>
    <Footer>
      <Span {...textDefaults} fontSize={13} color="#7d7d7d">This message was sent to <A {...textDefaults} textDecoration="none" href="mailto:{{.AccountEmailAddress}}">{'{{.AccountEmailAddress}}'}</A> because someone signed up for an account on <A {...textDefaults} textDecoration="none" href="{{.SiteURL}}">{'{{.SiteURLShort}}'}</A> with this email address. If you did not request this email, please disregard.</Span>
    </Footer>
  </StandardEmail>
)
