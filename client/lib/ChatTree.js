import _ from 'lodash'
import Immutable from 'immutable'

import Tree from './tree'


const initCount = Immutable.Map({
  descendants: 0,
  newDescendants: 0,
  ownDescendants: 0,
  mentionDescendants: 0,
  newMentionDescendants: 0,
  latestDescendantTime: null,
  latestDescendant: null,
})

const numericFields = initCount.filter(v => _.isNumber(v)).keySeq().cacheResult()

function mergeCount(origCount, newCount) {
  return origCount.withMutations(count => {
    numericFields.forEach(k => {
      count.set(k, count.get(k) + newCount.get(k))
    })

    if (newCount.get('latestDescendantTime') >= (count.get('latestDescendantTime') || 0)) {
      count.merge({
        latestDescendantTime: newCount.get('latestDescendantTime'),
        latestDescendant: newCount.get('latestDescendant'),
      })
    }
  })
}

function subtractCount(origCount, newCount) {
  // this only works for numeric counts
  return origCount.withMutations(count => {
    numericFields.forEach(k => {
      count.set(k, count.get(k) - newCount.get(k))
    })
  })
}

const DECAY = 10 * 60

class ChatTree extends Tree {
  constructor() {
    super('time')

    this.threads = new Tree('score')
    this.threads.reset()
  }

  _calculateCountScore(node, count) {
    if (node.get('children').size < 2) {
      return 0
    }
    const magnitude = 10 * count.get('mentionDescendants') + count.get('ownDescendants') + count.get('newDescendants') / 2 + count.get('descendants') / 10
    if (magnitude < 0.5) {
      return 0
    }
    const scaled = Math.log(Math.max(magnitude, 1)) / Math.LN2
    return scaled + count.get('latestDescendantTime') / DECAY
  }

  calculateNodeCount(node) {
    if (node === true || !node.get('parent')) {
      return initCount
    }
    return Immutable.Map({
      descendants: 1,
      newDescendants: +(!node.get('_seen') && !node.get('_own')),
      ownDescendants: +!!node.get('_own'),
      mentionDescendants: +!!node.get('_mention'),
      newMentionDescendants: +!!(node.get('_mention') && !node.get('_seen')),
      latestDescendantTime: node.get('time'),
      latestDescendant: node.get('id'),
    })
  }

  calculateDescendantCount(id, skip) {
    return initCount.withMutations(count => {
      Immutable.Seq(this.iterChildrenOf(id))
        .skip(skip || 0)
        .forEach(child => {
          const childDescendantCount = child.get('$count', initCount)
          mergeCount(count, childDescendantCount)
          const childNodeCount = this.calculateNodeCount(child)
          mergeCount(count, childNodeCount)
        })
    })
  }

  updateFunc(oldNodes, update) {
    // $count includes the aggregate values for descendants (excluding the node
    // itself). this simplifies the common use cases, such as displaying the
    // number of unread descendants. if the counts included the node itself, we
    // would have to subtract 1 from the 'newDescendants' property depending on
    // whether the node itself was seen.
    //
    // nodes that are orphans (or descendants of orphans) will be skipped (and
    // thus have no $count).
    //
    // TODO: when deletions (children changed) are someday desired, will need
    // the ability to mark ancestors to recalculate based on their children
    // instead of incremental update.

    const queue = _.pairs(oldNodes)
    const seen = _.mapValues(oldNodes, () => true)

    const scores = {}
    while (queue.length) {
      const [id, oldNode] = queue.shift()

      if (id.substr(0, 2) === '__') {
        continue
      }

      const ancestors = Immutable.List(this.iterAncestorsOf(id))
      if (!ancestors.size || ancestors.last().get('id') !== '__root') {
        // orphan
        continue
      }

      let node = this.index[id]
      if (!node.has('$count')) {
        // the node could have been an orphan (though it could also be new). we
        // should queue any child nodes for updating. either they are queued
        // already, or are orphan childs that now need counts.
        node.get('children')
          .filterNot(nid => _.has(seen, nid))  // eslint-disable-line no-loop-func
          .forEach(nid => queue.push([nid, true]))  // eslint-disable-line no-loop-func
        node = update(node.set('$count', initCount))
      }

      const oldNodeSelfCount = this.calculateNodeCount(oldNode)
      const nodeSelfCount = this.calculateNodeCount(node)

      if (Immutable.is(oldNodeSelfCount, nodeSelfCount)) {
        // if this node's counts are unchanged, we don't need to do anything
        continue
      }

      scores[id] = this._calculateCountScore(node, mergeCount(nodeSelfCount, node.get('$count')))

      const deltaCount = subtractCount(nodeSelfCount, oldNodeSelfCount)

      // walk ancestors, updating with this node's change in count
      ancestors.forEach(ancestor => {  // eslint-disable-line no-loop-func
        const ancestorId = ancestor.get('id')
        if (ancestorId === '__root') {
          return false
        }

        const ancestorDescendantCount = ancestor.get('$count', initCount)
        const updatedAncestorDescendantCount = mergeCount(ancestorDescendantCount, deltaCount)
        update(ancestor.set('$count', updatedAncestorDescendantCount))

        const ancestorCount = this.calculateNodeCount(ancestor)
        scores[ancestorId] = this._calculateCountScore(ancestor, mergeCount(ancestorCount, updatedAncestorDescendantCount))
      })
    }

    delete scores.__root
    this.updateThreads(scores)
  }

  updateThreads(scores) {
    const posScores = _.pick(scores, s => s > 0)
    if (!_.size(scores)) {
      return
    }

    const changedThreads = {}
    _.each(posScores, (origScore, threadId) => {
      let score = origScore
      const curThread = this.threads.get(threadId)
      let parentId

      if (curThread) {
        score = Math.max(score, -curThread.get('score'))
        parentId = curThread.get('parent')
      } else {
        parentId = Immutable.Seq(this.iterAncestorsOf(threadId))
          .map(ancestor => ancestor.get('id'))
          .find(ancestorId => _.has(posScores, ancestorId) || this.threads.get(ancestorId))

        // search for children of the parent that should now be children of the
        // current thread. we only need to check threads existing in the tree
        // because new threads will find the correct parent using the logic
        // above.
        if (this.threads.get(parentId)) {
          Immutable.Seq(this.threads.iterChildrenOf(parentId)).forEach(child => {
            const childId = child.get('id')
            const isChild = Immutable.Seq(this.iterAncestorsOf(childId))
              .some(ancestor => ancestor.get('id') === threadId)
            if (isChild) {
              changedThreads[childId] = _.extend(changedThreads[childId] || {id: childId}, {parent: threadId})
            }
          })
        }
      }

      changedThreads[threadId] = {
        id: threadId,
        parent: parentId,
        score: -score,
      }
    })

    this.threads.add(_.values(changedThreads))
  }

  getCount(id) {
    const node = this.index[id]
    if (!node) {
      return null
    }
    return node.get('$count', null)
  }

  reset(entries) {
    if (this.threads) {
      this.threads.reset()
    }
    return super.reset(entries)
  }
}

ChatTree.initCount = initCount
ChatTree.numericFields = numericFields
ChatTree.mergeCount = mergeCount
ChatTree.subtractCount = subtractCount
ChatTree.DECAY = DECAY

export default ChatTree  // work around https://github.com/babel/babel/issues/2694
