Contributing to Lit
============================

Thank you for deciding you want to contribute!

### Who is working on this?

As of this writing there are not many contributors yet, so things can be pretty chill regarding rules of contributions.

However, currently adiabat is the maintainer / lead dev and has full access to the repo. If you want to make a pull request, he'll have to approve it.

### What can I do?

- File issues you see that haven't been reported in the issues tab
- Make pull requests to fix these issues
- Add features (squash them if more than a few commits).  If it's a big change, discuss with people beforehand.  adiabat is on IRC in the various bitcoin-related channels.

### Code practices / philosophy

Imports are scary.  Standard library is better if possible.  More "not invented here" than "invented here" syndrome.

Go is in general pretty strict about formatting with go fmt, so as long as it's been go fmt'ed, there isn't too much to argue about.  That said:

Keep go code to 80 characters per line if feasible.

1 letter variable names are OK for "self" in a method, or iterators (i, j, k).  Other in-function variable names should be a bit more descriptive, and names of functions and fields which are exported should have descriptive, CamelCase names.
