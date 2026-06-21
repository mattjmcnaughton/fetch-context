## [1.0.1](https://github.com/mattjmcnaughton/fetch-context/compare/v1.0.0...v1.0.1) (2026-06-21)


### Bug Fixes

* **repoid:** clone SSH refs over SSH instead of HTTPS ([c03c0a6](https://github.com/mattjmcnaughton/fetch-context/commit/c03c0a6829b2b41ffa198a9dd2873555fe9087d6))

# 1.0.0 (2026-06-13)


### Features

* **adapters:** gitfixture server and the gitrepo/hostrepo/filestore adapters ([123ab30](https://github.com/mattjmcnaughton/fetch-context/commit/123ab30567c457b089fb1adf3ec350ba7731fcb2))
* **cli:** repo subcommand, wiring, and the Phase-2 e2e suite ([8ed5c8a](https://github.com/mattjmcnaughton/fetch-context/commit/8ed5c8a8ade8bc788e002181ed1e147be7d1b285))
* **cli:** walking skeleton — version subcommand and R1 exit-code mapping ([80487ee](https://github.com/mattjmcnaughton/fetch-context/commit/80487ee2e47ad4eabf71b7202161e5cb203916e6))
* **config:** clone section and polymorphic repo entries ([53b2320](https://github.com/mattjmcnaughton/fetch-context/commit/53b232015502a9c6dbb7f959e951af08cacc3eab))
* **contract:** close the loop — real-API contract twins, doc verification ([6ec4044](https://github.com/mattjmcnaughton/fetch-context/commit/6ec40441de9bebb59e454c6e60264856546b341a))
* **core:** bounded parallel runner + ADR-0002 ([ce94a93](https://github.com/mattjmcnaughton/fetch-context/commit/ce94a93814d4223b81fd460181452e423c4f1c43))
* **core:** MaterializeRepo use case with first-wave ports and fakes ([aa833c6](https://github.com/mattjmcnaughton/fetch-context/commit/aa833c693999fceef78d5004427b31009fd92d9d))
* **core:** repoid normalization (R6) and target path mapping ([9c7e706](https://github.com/mattjmcnaughton/fetch-context/commit/9c7e70648c16ca788dfc86d8085bbd3261a4c353))
* **gitrepo:** CloneOptions on the port; depth, branch, and converging refresh ([5932b5f](https://github.com/mattjmcnaughton/fetch-context/commit/5932b5f15f229ccfd85ae63f21fac01426c5db23))
* **group:** --depth for enumerated clones ([46eb1b1](https://github.com/mattjmcnaughton/fetch-context/commit/46eb1b1cea5bd3c84e2d745784bf1125de062cd4))
* **group:** group slice — forge enumeration, dispatch-by-host, e2e ([e1b4407](https://github.com/mattjmcnaughton/fetch-context/commit/e1b4407b4727526c36fdee46f935914dedf5187f))
* **parallel:** bounded-parallel clones in group, repo, and load ([d8e1fa1](https://github.com/mattjmcnaughton/fetch-context/commit/d8e1fa13dc9a19626932b091ce94443c81d7b967))
* **profiles:** config store, load/list/clean/edit slices, e2e ([7c2df21](https://github.com/mattjmcnaughton/fetch-context/commit/7c2df219c4eade936db1caa138b8b214f7ff6308))
* **release:** semantic-release pipeline with multi-arch binaries ([69329de](https://github.com/mattjmcnaughton/fetch-context/commit/69329de7c2987ebd9d36100425bb3214670f8e16))
* **repo:** --depth/--branch flags and per-entry clone options ([12e45dc](https://github.com/mattjmcnaughton/fetch-context/commit/12e45dc6a846f845b898e0282c980032cbe60cb8))
* **url:** url slice — R5 mapping, PageReader port, adapter, CLI, e2e ([de48ec9](https://github.com/mattjmcnaughton/fetch-context/commit/de48ec96956907485895f7b5abd0036ddffd15b4))
