// ignore Text, Image redefinition
// jshint -W079

var React = require('react')

var email = require('./email')
var Item = email.Item
var Text = email.Text
var Link = email.Link
var common = require('./common')
var textDefaults = common.textDefaults


module.exports = (
  <common.StandardEmail>
    <common.TopBubbleBox logo="logo-warning.png" padding={15}>
      <Item align="center">
        <Text {...textDefaults} fontSize={24}>your password has been changed</Text>
      </Item>
    </common.TopBubbleBox>
    <common.BodyBox>
      <Item>
        <Text {...textDefaults}>hey {'{{AccountName}}'}, just keeping you in the loop. if you just updated your <Link {...textDefaults} href="{{SiteURL}}">{'{{SiteName}}'}</Link> password, you're good to go!</Text>
      </Item>
      <Item>
        <Text {...textDefaults}>if you did not change your password and suspect something fishy is going on, please reply to this email immediately.</Text>
      </Item>
    </common.BodyBox>
    <common.Footer>
      {common.standardFooter}
    </common.Footer>
  </common.StandardEmail>
)
