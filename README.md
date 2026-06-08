# no-lsp

## DISCLAIMER: This project is in an early exploratory MVP stage. Nothing is final, and the code is ugly.

### What is it?

An attempt to recreate "pre-LSP" workflows for LSP-based editors. At the moment, it includes:

* The `edit → build → fix` workflow inspired by [Vim's quickfix](https://vimdoc.sourceforge.net/htmldoc/quickfix.html#quickfix) and [Sublime Text's Build Mode](https://www.sublimetext.com/docs/build_systems.html#basic-example);
* A way to specify additional source directories for inspecting standard library and third-party library code.

### Why?

LSPs are large, complex, and vary significantly in both feature set and stability. 

There may be value in having a single, robust one that provides a consistent experience and covers arguably 95% of day-to-day development needs across virtually any stack/codebase in LSP-first editors (Helix, Visual Studio Code, Zed etc.).


