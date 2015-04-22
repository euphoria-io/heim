var _ = require('lodash')
var Immutable = require('immutable')

var Tree = require('./tree')


var initCount = Immutable.Map({
  descendants: 0,
  newDescendants: 0,
  ownDescendants: 0,
  mentionDescendants: 0,
  newMentionDescendants: 0,
  latestDescendantTime: null,
  latestDescendant: null,
})

var numericFields = initCount.filter(v => _.isNumber(v)).keySeq().cacheResult()

function mergeCount(count, newCount) {
  count = count.withMutations(count => {
    numericFields.forEach(k => {
      count.set(k, count.get(k) + newCount.get(k))
    })
  })
  if (newCount.get('latestDescendantTime') >= (count.get('latestDescendantTime') || 0)) {
    count = count.merge({
      latestDescendantTime: newCount.get('latestDescendantTime'),
      latestDescendant: newCount.get('latestDescendant'),
    })
  }
  return count
}

function subtractCount(count, newCount) {
  // this only works for numeric counts
  count = count.withMutations(count => {
    numericFields.forEach(k => {
      count.set(k, count.get(k) - newCount.get(k))
    })
  })
  return count
}

var DECAY = 10 * 60

var ChatTree = module.exports = function() {
  this.threads = new Tree('score')
  Tree.call(this, 'time', this.updateCounts)
}

ChatTree.prototype = _.create(Tree.prototype, {
  _calculateCountScore: function(node, count) {
    if (node.get('children').size < 2) {
      return 0
    }
    var magnitude = 10 * count.get('mentionDescendants') + count.get('ownDescendants') + count.get('newDescendants') / 2 + count.get('descendants') / 10
    if (magnitude < 0.5) {
      return 0
    }
    var scaled = Math.log(Math.max(magnitude, 1)) / Math.LN2
    return scaled + count.get('latestDescendantTime') / DECAY
  },

  calculateNodeCount: function(node) {
    if (node === true || !node.get('parent')) {
      return initCount
    } else {
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
  },

  calculateDescendantCount: function(id, skip) {
    return initCount.withMutations(count => {
      Immutable.Seq(this.iterChildrenOf(id))
        .skip(skip || 0)
        .forEach(child => {
          var childDescendantCount = child.get('$count', initCount)
          mergeCount(count, childDescendantCount)
          var childNodeCount = this.calculateNodeCount(child)
          mergeCount(count, childNodeCount)
        })
    })
  },

  updateCounts: function(oldNodes, update) {
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

    var queue = _.pairs(oldNodes)
    var seen = _.mapValues(oldNodes, () => true)

    var scores = {}
    while (queue.length) {
      // TODO: es6
      var entry = queue.shift()
      var id = entry[0]
      var oldNode = entry[1]

      if (id.substr(0, 2) == '__') {
        continue
      }

      var ancestors = Immutable.List(this.iterAncestorsOf(id))
      if (!ancestors.size || ancestors.last().get('id') != '__root') {
        // orphan
        continue
      }

      var node = this.index[id]
      if (!node.has('$count')) {
        // the node could have been an orphan (though it could also be new). we
        // should queue any child nodes for updating. either they are queued
        // already, or are orphan childs that now need counts.
        node.get('children')
          .filterNot(id => _.has(seen, id))
          .forEach(id => queue.push([id, true]))
        node = update(node.set('$count', initCount))
      }

      var oldNodeSelfCount = this.calculateNodeCount(oldNode)
      var nodeSelfCount = this.calculateNodeCount(node)

      if (Immutable.is(oldNodeSelfCount, nodeSelfCount)) {
        // if this node's counts are unchanged, we don't need to do anything
        continue
      }

      scores[id] = this._calculateCountScore(node, mergeCount(nodeSelfCount, node.get('$count')))

      var deltaCount = subtractCount(nodeSelfCount, oldNodeSelfCount)

      // walk ancestors, updating with this node's change in count
      ancestors.forEach(ancestor => {
        var ancestorId = ancestor.get('id')
        if (ancestorId == '__root') {
          return false
        }

        var ancestorDescendantCount = ancestor.get('$count', initCount)
        var updatedAncestorDescendantCount = mergeCount(ancestorDescendantCount, deltaCount)
        update(ancestor.set('$count', updatedAncestorDescendantCount))

        var ancestorCount = this.calculateNodeCount(ancestor)
        scores[ancestorId] = this._calculateCountScore(ancestor, mergeCount(ancestorCount, updatedAncestorDescendantCount))
      })
    }

    delete scores.__root
    this.updateThreads(scores)
  },

  updateThreads: function(scores) {
    scores = _.pick(scores, s => s > 0)
    if (!_.size(scores)) {
      return
    }

    var changedThreads = {}
    _.each(scores, (score, threadId) => {
      var curThread = this.threads.get(threadId)
      var parentId

      if (curThread) {
        score = Math.max(score, -curThread.get('score'))
        parentId = curThread.get('parent')
      } else {
        parentId = Immutable.Seq(this.iterAncestorsOf(threadId))
          .map(ancestor => ancestor.get('id'))
          .find(ancestorId => _.has(scores, ancestorId) || this.threads.get(ancestorId))

        // search for children of the parent that should now be children of the
        // current thread. we only need to check threads existing in the tree
        // because new threads will find the correct parent using the logic
        // above.
        if (this.threads.get(parentId)) {
          Immutable.Seq(this.threads.iterChildrenOf(parentId)).forEach(child => {
            var childId = child.get('id')
            var isChild = Immutable.Seq(this.iterAncestorsOf(childId))
              .some(ancestor => ancestor.get('id') == threadId)
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
  },

  getCount: function(id) {
    var node = this.index[id]
    if (!node) {
      return null
    }
    return node.get('$count', null)
  },

  reset: function() {
    this.threads.reset()
    Tree.prototype.reset.apply(this, arguments)
    return this
  },
})

ChatTree.initCount = initCount
ChatTree.numericFields = numericFields
ChatTree.mergeCount = mergeCount
ChatTree.subtractCount = subtractCount
ChatTree.DECAY = DECAY
