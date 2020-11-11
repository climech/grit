# Grit #

Grit is a personal task management tool which regards tasks as nodes in a [directed acyclic graph](https://en.wikipedia.org/wiki/Directed_acyclic_graph), rather than elements in a traditional to-do list. The graph structure enables subdivision of tasks and seamless integration between short-term and long-term goals.

## Installation ##

Make sure you have both `go` and `gcc` installed, then:

```
$ git clone https://github.com/climech/grit.git
$ cd grit/
$ make && sudo make install
```

**Note:** Grit is still in very early stages. Things may break, so it's a good idea to make backups of `~/.config/grit`.

## How it works ##

Grit is based on two premises:

1. Breaking up a problem into smaller, more manageable parts and tackling them one by one is generally a good approach to problem-solving; if needed, those parts can be further subdivided into yet smaller ones, and so on.
2. Keeping track of your progress across time improves focus and motivation, especially with many parallel tasks spanning multiple days.

A bigger task can be represented by a tree, e.g.:

```
[~] Digitize family photos
 ├──[x] Scan album 1
 ├──[x] Scan album 2
 ├──[ ] Scan album 3
 └──[ ] ...
```

In this arrangement, completing all the subtasks of a task is equivalent to completing the task. It becomes useful to allow cross edges between nodes belonging to different trees (see _Links_), so we use a more general structure of the directed acyclic graph (DAG) in which the arrows point from parent tasks (predecessors) to their subtasks (successors):

![Figure 1](https://i.imgur.com/IQkHFIC.png)

A **date node** (or **d-node**) is a special case of node which enables us to associate tasks with specific dates. They are identified by their names, which are dates in the format `YYYY-MM-DD`. Date nodes exist so long as there are successors connected to them. They are created and destroyed automatically.

Grit tasks exist in one of the three possible states: _inactive_ (` `), _in progress_ (`~`) and _completed_ (`x`). _Inactive_ means the task hasn't been started yet. _In progress_ indicates that some of the task's subtasks have been completed.

## A practical guide ##

### A simple list ###

Let's add a few things we want to do today:

```
$ grit add "Take out the trash"
[++] Created node: (1) -> [ ] Take out the trash (2)
$ grit add "Do the laundry"
[++] Created node: (1) -> [ ] Do the laundry (3)
$ grit add "Call Dad"
[++] Created node: (1) -> [ ] Call Dad (4)
```

Run `grit` without arguments to display the current date node.

```
$ grit
[ ] 2020-11-10 (1)
 ├──[ ] Take out the trash (2)
 ├──[ ] Do the laundry (3)
 └──[ ] Call Dad (4)
```

So far it looks like an old-fashioned to-do list. We can mark tasks as completed using the `check` command.

```
$ grit check 2
$ grit
[~] 2020-11-10 (1)
 ├──[x] Take out the trash (2)
 ├──[ ] Do the laundry (3)
 └──[ ] Call Dad (4)
```

The change is automatically propagated through the graph. We can see that the status of the date node has changed to _in progress_ (~).

### Adding successors ###

Let's add another task.

```
$ grit add "Get groceries"
[++] Created node: (1) -> [ ] Get groceries (5)
$ grit
[~] 2020-11-10 (1)
 ├──[x] Take out the trash (2)
 ├──[ ] Do the laundry (3)
 ├──[ ] Call Dad (4)
 └──[ ] Get groceries (5)
```

Say we want to break it up into smaller pieces. In Grit, this is equivalent to adding successors to the node. We can do this by specifying the predecessor (or parent) with the `-p` flag.

```
$ grit add -p 5 "Bread"
[++] Created node: (5) -> [ ] Bread (6)
$ grit add -p 5 "Milk"
[++] Created node: (5) -> [ ] Milk (7)
$ grit add -p 5 "Eggs"
[++] Created node: (5) -> [ ] Eggs (8)
```

Now we have task 5 pointing to subtasks 6, 7 and 8. We can go infinitely deep if needed.

```
$ grit
[~] 2020-11-10 (1)
 ├──[x] Take out the trash (2)
 ├──[ ] Do the laundry (3)
 ├──[ ] Call Dad (4)
 └──[ ] Get groceries (5)
     ├──[ ] Bread (6)
     ├──[ ] Milk (7)
     └──[ ] Eggs (8)
```

Check the whole branch:

```
$ grit check 5
$ grit tree 5
[x] Do shopping (5)
 ├──[x] Get rice (6)
 ├──[x] Get hand sanitizer (7)
 └──[x] Get toilet paper (8)
```

The `tree` command prints a pretty tree representation rooted at the given node. When we run `grit` without arguments, `tree` is executed implicitly, defaulting to the current d-node.

### Adding roots ###

So far we've only added successors to other nodes—`add` adds successors to the current d-node by default.

Say we have a bigger task to complete—work through an Algebra textbook. This will definitely take more than one day to complete, so we can't simply make it a successor of a single date node.

Let's create a new root node, then. Books are already structured like trees, making it easier for us. Our book is divided into 35 chapters, each of which is divided into smaller sections and example exercises. To create a root, add `-r` before the name.

```
$ grit add -r "Work through Higher Algebra by H. S. Hall, S. R. Knight"
[++] Created root: [ ] Work through Higher Algebra by H. S. Hall, S. R. Knight (9)
```

We will be referencing this node a lot, so it's a good idea to give it an alias.

```
$ grit alias 5 textbook
```

Adding the chapters one by one would be very laborious. Let's use a Bash loop to make the job easier.

```
$ for i in {1..35}; do grit add -p textbook "Chapter $i"; done
[++] Created node: (9) -> [ ] Chapter 1 (10)
[++] Created node: (9) -> [ ] Chapter 2 (11)
[++] Created node: (9) -> [ ] ...
[++] Created node: (9) -> [ ] Chapter 35 (44)
```

Working through a chapter involves reading all the sections and solving all the exercises it contains. Chapter 1 has 28 exercises, numbered 1-28. I like to have them all as separate nodes—this way I can solve them over a number of days without losing track, while I work on other chapters.

```
$ grit add -p 10 "Read the chapter"
[++] Created node: (10) -> [ ] Read the chapter (45)
$ grit add -p 10 "Solve the exercises"
[++] Created node: (10) -> [ ] Solve the exercises (46)
$ for i in {1..28}; do grit add -p 46 "Solve ex. $i"; done
[++] Created node: (46) -> [ ] Solve ex. 1 (47)
[++] Created node: (46) -> [ ] Solve ex. 2 (48)
[++] Created node: (46) -> ...
[++] Created node: (46) -> [ ] Solve ex. 28 (74)
```

Our tree so far:

```
$ grit tree textbook
[ ] Work through Higher Algebra by H. S. Hall, S. R. Knight (9:textbook)
 ├──[ ] Chapter 1 (10)
 │   ├──[ ] Read the chapter (45)
 │   └──[ ] Solve the exercises (46)
 │       ├──[ ] Solve ex. 1 (47)
 │       ├──[ ] Solve ex. 2 (48)
 │       ├──[ ] ...
 │       └──[ ] Solve ex. 28 (74)
 ├──[ ] Chapter 2 (11)
 ├──[ ] Chapter ...
 └──[ ] Chapter 35 (44)
```

We can create the whole tree this way, or add branches later as we go along.

Run `stat` to display more information about the node:

```
$ grit stat textbook

(9) ───┬─── (10)
       ├─── ...
       └─── (44)

ID: 9
Name: Work through Higher Algebra by H. S. Hall, S. R. Knight
Status: inactive (0/63)
Predecessors: 0
Successors: 35
Alias: textbook
```

The current progress is calculated by counting the leaves reachable from the node.

### Links ###

Say we want to work on the first chapter of our Algebra book today. Let's add a new task to the current d-node.

```
$ grit add "Work on ch. 1 of the Algebra textbook"
[++] Created node: (1) -> [ ] Work on ch. 1 of the Algebra textbook (75)
```

We can link this node to the relevant subtasks of `textbook`.

```
$ grit link 75 45
[++] Created edge: (75) -> (45)
$ grit link 75 47
[++] Created edge: (75) -> (47)
$ grit link 75 48
[++] Created edge: (75) -> (48)
$ grit link 75 49
[++] Created edge: (75) -> (49)
```

Now we have nodes with multiple parents, highlighted in the figure below. We've turned our trees into a proper digraph!

![Figure 2](https://i.imgur.com/2efpjvQ.png)

The connected nodes show up in the current d-node view:

```
$ grit
[~] 2020-11-10 (1)
 ├──[x] ...
 └──[ ] Work on ch. 1 of the Algebra textbook (75)
     ├──[ ] Read the chapter (45)
     ├──[ ] Solve ex. 1 (47)
     ├──[ ] Solve ex. 2 (48)
     └──[ ] Solve ex. 3 (49)
```

We can also run `stat` to confirm that the nodes have two parents (predecessors):

```
$ grit stat 45

(10) ───┐
(75) ───┴─── (45)

ID: 45
Name: Read the chapter
Status: inactive
Predecessors: 2
Successors: 0
```

Checking the tasks will be reflected in all predecessor views.

```
$ grit check 75
$ grit
[x] 2020-11-10 (1)
 ├──[x] ...
 └──[x] Work on ch. 1 of the Algebra textbook (75)
     ├──[x] Read the chapter (45)
     ├──[x] Solve ex. 1 (47)
     ├──[x] Solve ex. 2 (48)
     └──[x] Solve ex. 3 (49)
```

The d-node is now completed, but there's still more work to be done for `textbook`:

```
$ grit tree textbook
[~] Work through Higher Algebra by H. S. Hall, S. R. Knight (9:textbook)
 ├──[~] Chapter 1 (10)
 │   ├──[x] Read the chapter (45)
 │   └──[~] Solve the exercises (46)
 │       ├──[x] Solve ex. 1 (47)
 │       ├──[x] Solve ex. 2 (48)
 │       ├──[x] Solve ex. 3 (49)
 │       ├──[ ] Solve ex. 4 (50)
 │       ├──[ ] ...
 │       └──[ ] Solve ex. 28 (74)
 ├──[ ] ...
 └──[ ] Chapter 35 (44)
```

Again, `stat` will tell us how much progress we've made so far:

```
$ grit stat textbook

(9) ───┬─── (10)
       ├─── ...
       └─── (44)

ID: 9
Name: Work through Higher Algebra by H. S. Hall, S. R. Knight
Status: in progress (4/63)
Predecessors: 0
Successors: 35
Alias: textbook
```

### Emerging patterns: a reading challenge ###

Giving yourself a challenge can be a nice motivational tool. Let's say our goal is to read 25 books this year. We start with the root task:

```
$ grit add -r "Challenge: Read 25 books in 2020"
[++] Created root: [ ] Challenge: Read 25 books in 2020 (76)
$ grit alias 76 rc2020
```

We could stop here, and add the books to it as we go, but this alone won't give us a nice way to check our progress. Let's go a little further and create a "slot" for each of the 25 books. We'll be able to link these slots to the relevant daily tasks. To create the slots, we'll again use a Bash loop:

```
$ for i in {1..25}; do grit add -p rc2020 "Book $i"; done
[++] Created node: (76) -> [ ] Book 1 (77)
[++] Created node: (76) -> [ ] Book 2 (78)
[++] Created node: (76) -> [ ] ...
[++] Created node: (76) -> [ ] Book 25 (101)
```

Now we have a nice tree:
```
$ grit tree rc2020
[ ] Challenge: Read 25 books in 2020 (76:rc2020)
 ├──[ ] Book 1 (77)
 ├──[ ] Book 2 (78)
 ├──[ ] ...
 └──[ ] Book 25 (101)
```

We can link the "slots" to the specific books we read during the year:

```
$ grit add "Read 1984 by G. Orwell"
[++] Created node: (1) -> [ ] Read 1984 by G. Orwell (102)
$ grit link 77 102
[++] Created edge: (77) -> (102)
$ grit check 102
```

Check our progress:

```
$ grit stat rc2020

(76) ───┬─── (77)
        ├─── ...
        └─── (101)

ID: 76
Name: Challenge: Read 25 books in 2020
Status: in progress (1/25)
Predecessors: 0
Successors: 25
Alias: rc2020
```

-----------

© 2020 climech.org