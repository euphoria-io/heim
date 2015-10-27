var React = require('react')

var common = require('./common')
var heimURL = require('../lib/heim-url')


module.exports = (
  <common.MainPage title="about euphoria" className="about">
    <div className="primary">
      <h1>what's euphoria?</h1>
      <section className="letter">
        <h3>Euphoria is chat for all, in communities that matter.</h3>
        <p>Hello! We're building a platform for chat rooms you care about.</p>
        <p>Social chat rooms have played an important role in each of our lives. They’ve entertained us, offered inspiration and feedback, and given us lasting friends and relationships.</p>
        <p>Our goal is to foster new spaces that can offer this experience to you. Here's a bit more about us. We look forward to learning more about you.</p>
        <div className="end">
          <p className="contact">
            <span className="label">Got a question, or want to learn more?</span>
            <a className="chat-with-us" href={heimURL('/room/welcome/')}>chat with us &raquo;</a>
          </p>
          <p className="signature">
            &mdash; <common.FauxNick nick="intortus" />, <common.FauxNick nick="chromakode" />, and <common.FauxNick nick="greenie" />
          </p>
        </div>
      </section>
    </div>
    <section>
      <h2>our values (in emoji form)</h2>
      <common.FancyLogo />
      <ul className="values">
        <li className="welcoming">
          <h4>euphoria is welcoming</h4>
          <p>friendly to newcomers, easy to join, and safe.</p>
        </li>
        <li className="diverse">
          <h4>euphoria is diverse</h4>
          <p>we strive for tolerance, fairness, and accessibility.</p>
        </li>
        <li className="meaningful">
          <h4>euphoria is informal yet meaningful</h4>
          <p>everyone deserves respect, empathy, and understanding.</p>
        </li>
      </ul>
      <p>for more details, check out <a href={heimURL('/about/values')}>Euphoria's Values</a> statement.</p>
    </section>
    <section>
      <h2>euphoria is open</h2>
      <h3>We believe that online community platforms should be open source.</h3>
      <p>Our chat server, Heim, is <a href="https://github.com/euphoria-io/heim">available on GitHub</a>. Join our development chat in <a href={heimURL('/room/heim')}>&heim</a>.</p>
    </section>
    <section className="who">
      <h2>who we are</h2>
      <h3>We're a small team who care deeply about online socialization and citizenship.</h3>
      <div className="messages wrap">
        <common.FauxMessage sender="intortus" message="hi, I'm logan. I'm an erstwhile motorcycle racer and one of euphoria's programmers. you can meet new users with me in &welcome or discuss backend development with me in &heim." />
        <common.FauxMessage sender="chromakode" message="hey, I'm Max! I live in San Francisco, where I work on Euphora's user interface and design. you can often find me jamming in &music or working in &heim." />
        <common.FauxMessage sender="greenie" message="oh hai, I’m Kris. I live with too many cats in the deep forest of Vermont. when I’m not busy getting into moose-caused traffic jams, I do community stuff for Euphoria. I tend to be found linking articles in &space and playing bluegrass tunes in &music." />
        <common.FauxMessage sender="ezzie" message="hi, I'm ezzie, the office dog!" embed={heimURL('/static/ezzie.jpg')}>
          <div className="replies">
            <common.FauxMessage sender="ezzie" message="you can find more photos of me in the room &ezziethedog." />
          </div>
        </common.FauxMessage>
      </div>
    </section>
  </common.MainPage>
)
