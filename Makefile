include Makefile.defs

.PHONY: all
all:

# ----------------

define BUILD_BIN
echo "begin to build bin for $(CMD_BIN_DIR)" ; mkdir -p $(DESTDIR_BIN) ; \
   BIN_NAME_LIST=$$( cd $(CMD_BIN_DIR) && ls ) ; \
   for BIN_NAME in $${BIN_NAME_LIST} ; do \
  		rm -f $(DESTDIR_BIN)/$${BIN_NAME} ; \
  		$(GO_BUILD) -o $(DESTDIR_BIN)/$${BIN_NAME}  $(CMD_BIN_DIR)/$${BIN_NAME}/main.go ; \
  		(($$?!=0)) && echo "error, failed to build $${BIN_NAME}" && exit 1 ; \
  		echo "succeeded to build $${BIN_NAME} to $(DESTDIR_BIN)/$${BIN_NAME}" ; \
  	 done
endef

.PHONY: build-netknife-bin
build-netknife-bin: CMD_BIN_DIR:=netknifeCmd
build-netknife-bin:
	@ $(BUILD_BIN)


.PHONY: build-smartdocker-bin
build-smartdocker-bin: CMD_BIN_DIR:=smartdocterCmd
build-smartdocker-bin:
	@ $(BUILD_BIN)


# ==========================


define BUILD_FINAL_IMAGE
echo "Build Image with tag: $(GIT_COMMIT_VERSION)" ; \
		docker buildx build  \
				--build-arg GIT_COMMIT_VERSION=$(GIT_COMMIT_VERSION) \
				--build-arg GIT_COMMIT_TIME=$(GIT_COMMIT_TIME) \
				--build-arg VERSION=$(GIT_COMMIT_VERSION) \
				--file $(ROOT_DIR)/images/"$${FINAL_IMAGES##*/}"/Dockerfile \
				--output type=docker \
				--tag $${FINAL_IMAGES}:$(GIT_COMMIT_VERSION) . ; \
		echo "build success for $${i}:$(GIT_COMMIT_VERSION) "
endef


.PHONY: build_local_image
build_local_image: build_local_netknife_image


.PHONY: build_local_netknife_image
build_local_netknife_image: FINAL_IMAGES := ${REGISTER}/${GIT_REPO}/netknife
build_local_netknife_image:
	@ $(BUILD_FINAL_IMAGE)


#---------

.PHONY: build_local_base_image
build_local_base_image: build_local_netknife_base_image


define BUILD_BASE_IMAGE
TAG=` git ls-tree --full-tree HEAD -- $(IMAGEDIR) | awk '{ print $$3 }' ` ; \
		echo "Build base image with tag: $${TAG}" ; \
		docker buildx build  \
				--build-arg USE_PROXY_SOURCE=true \
				--file $(IMAGEDIR)/Dockerfile \
				--output type=docker \
				--tag $(BASE_IMAGES):$${TAG}  $(IMAGEDIR) ; \
		(($$?==0)) || { echo "error , failed to build base image" ; exit 1 ;} ; \
		echo "build success $(BASE_IMAGES):$${TAG} "
endef

.PHONY: build_local_netknife_base_image
build_local_netknife_base_image: IMAGEDIR := ./images/netknife-base
build_local_netknife_base_image: BASE_IMAGES := ${REGISTER}/${GIT_REPO}/netknife-base
build_local_netknife_base_image:
	@ $(BUILD_BASE_IMAGE)


# ==========================

.PHONY: package-charts
package-charts:
	@ make -C charts package

.PHONY: lint-golang
lint-golang: LINT_DIR := ./netknifeCmd/...
lint-golang:
	$(QUIET) tools/check-go-fmt.sh
	$(QUIET) $(GO_VET)  $(LINT_DIR)
	$(QUIET) golangci-lint run



.PHONY: lint-markdown-format
lint-markdown-format:
	@$(CONTAINER_ENGINE) container run --rm \
		--entrypoint sh -v $(ROOT_DIR):/workdir ghcr.io/igorshubovych/markdownlint-cli:latest \
		-c '/usr/local/bin/markdownlint -c /workdir/.github/markdownlint.yaml -p /workdir/.github/markdownlintignore  /workdir/' ; \
		if (($$?==0)) ; then echo "congratulations ,all pass" ; else echo "error, pealse refer <https://github.com/DavidAnson/markdownlint/blob/main/doc/Rules.md> " ; fi


.PHONY: fix-markdown-format
fix-markdown-format:
	@$(CONTAINER_ENGINE) container run --rm  \
		--entrypoint sh -v $(ROOT_DIR):/workdir ghcr.io/igorshubovych/markdownlint-cli:latest \
		-c '/usr/local/bin/markdownlint -f -c /workdir/.github/markdownlint.yaml -p /workdir/.github/markdownlintignore  /workdir/'



.PHONY: lint-yaml
lint-yaml:
	@$(CONTAINER_ENGINE) container run --rm \
		--entrypoint sh -v $(ROOT_DIR):/data cytopia/yamllint \
		-c '/usr/bin/yamllint -c /data/.github/yamllint-conf.yml /data' ; \
		if (($$?==0)) ; then echo "congratulations ,all pass" ; else echo "error, pealse refer <https://yamllint.readthedocs.io/en/stable/rules.html> " ; fi


.PHONY: unitest-tests
unitest-tests: UNITEST_DIR := netknifeCmd
unitest-tests:
	@echo "run unitest-tests"
	$(QUIET) $(ROOT_DIR)/tools/ginkgo.sh   \
		--cover --coverprofile=./coverage.out --covermode set  \
		--json-report unitestreport.json \
		-randomize-suites -randomize-all --keep-going  --timeout=1h  -p   --slow-spec-threshold=120s \
		-vv  -r   $(UNITEST_DIR)
	$(QUIET) go tool cover -html=./coverage.out -o coverage-all.html
