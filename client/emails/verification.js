import React from 'react'

import { Item, Span } from 'react-html-email'
import { StandardEmail, TopBubbleBox, BigButton, BodyBox, standardFooter, textDefaults } from './common'


module.exports = (
  <StandardEmail>
    <TopBubbleBox logo="logo.png" padding={15}>
      <Item align="center">
        <Span {...textDefaults} fontSize={20}>Please verify your email address:</Span>
      </Item>
    </TopBubbleBox>
    <BodyBox>
      <Item align="center">
        <Span {...textDefaults}>We're received a request to use this email address for your {'{{.SiteName}}'} account:</Span>
      </Item>
      <BigButton color="#80c080" href="{{.VerifyEmailURL}}">
        verify your email address
      </BigButton>
      <Item>
        <Span {...textDefaults}>If you did not make this request, please disregard this message. Thanks!</Span>
      </Item>
    </BodyBox>
    {standardFooter}
  </StandardEmail>
)
