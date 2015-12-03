import React from 'react'

import { Item, Text, Link } from './email'
import { StandardEmail, TopBubbleBox, BodyBox, BigButton, standardFooter, textDefaults } from './common'


module.exports = (
  <StandardEmail>
    <TopBubbleBox logo="logo-warning.png" padding={15}>
      <Item align="center">
        <Text {...textDefaults} fontSize={20}>would you like to reset your password?</Text>
      </Item>
    </TopBubbleBox>
    <BodyBox>
      <Item>
        <Text {...textDefaults}>hey, we've received a password reset request for your <Link {...textDefaults} href="{{.SiteURL}}">{'{{.SiteName}}'}</Link> account:</Text>
      </Item>
      <BigButton color="#dca955" href="{{.ResetPasswordURL}}">
        reset your password
      </BigButton>
      <Item>
        <Text {...textDefaults}>if you did not make this request and suspect something fishy is going on, please reply to this email immediately.</Text>
      </Item>
    </BodyBox>
    {standardFooter}
  </StandardEmail>
)
