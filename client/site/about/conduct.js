import fs from 'fs'
import React from 'react'

import { MainPage, PolicyNav, Markdown } from '../common'


module.exports = (
  <MainPage title="euphoria: conduct" nav={<PolicyNav selected="conduct" />}>
    <Markdown className="policy" content={fs.readFileSync(__dirname + '/conduct.md', 'utf8')} />
  </MainPage>
)
