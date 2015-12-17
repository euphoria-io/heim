import React from 'react'

import { Item, Text } from './email'
import { StandardEmail, TopBubbleBox, BigButton, BodyBox, standardFooter, textDefaults } from './common'


module.exports = (
  <StandardEmail>
    <TopBubbleBox logo="logo.png" padding={15}>
      <Item align="center">
        <Text {...textDefaults} fontSize={20}>Please verify your email address:</Text>
      </Item>
    </TopBubbleBox>
    <BodyBox>
      <Item align="center">
        <Text {...textDefaults}>We're received a request to use this email address for your {'{{.SiteName}}'} account:</Text>
      </Item>
      <BigButton color="#80c080" href="{{.VerifyEmailURL}}">
        verify your email address
      </BigButton>
      <Item>
        <Text {...textDefaults}>If you did not make this request, please disregard this message. Thanks!</Text>
      </Item>
    </BodyBox>
    {standardFooter}
  </StandardEmail>
)
