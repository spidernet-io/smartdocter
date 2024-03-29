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

.PHONY: build_smartdocter_agent_bin
build_smartdocter_agent_bin: CMD_BIN_DIR:=smartdocter-agent-cmd
build_smartdocter_agent_bin:
	@ $(BUILD_BIN)


.PHONY: build_smartdocter_controller_bin
build_smartdocter_controller_bin: CMD_BIN_DIR:=smartdocter-controller-cmd
build_smartdocter_controller_bin:
	@ $(BUILD_BIN)


# ==========================


define BUILD_FINAL_IMAGE
echo "Build Image with tag: $(IMAGE_TAG)" ; \
		docker buildx build  \
				--build-arg GIT_COMMIT_VERSION=$(GIT_COMMIT_VERSION) \
				--build-arg GIT_COMMIT_TIME=$(GIT_COMMIT_TIME) \
				--build-arg VERSION=$(GIT_COMMIT_VERSION) \
				--file $(IMAGEDIR)/Dockerfile \
				--output type=docker \
				--tag ${FINAL_IMAGES}:$(IMAGE_TAG) . ; \
		echo "build success for $${i}:$(IMAGE_TAG) "
endef


.PHONY: build_local_image
build_local_image: build_local_smartdocter_agent_image build_local_smartdocter_controller_image

.PHONY: build_local_smartdocter_agent_image
build_local_smartdocter_agent_image: FINAL_IMAGES := $(IMAGE_NAME_AGENT)
build_local_smartdocter_agent_image: IMAGEDIR := $(ROOT_DIR)/images/smartdocter-agent
build_local_smartdocter_agent_image: IMAGE_TAG := $(GIT_COMMIT_VERSION)
build_local_smartdocter_agent_image:
	@ $(BUILD_FINAL_IMAGE)

.PHONY: build_local_smartdocter_controller_image
build_local_smartdocter_controller_image: FINAL_IMAGES := $(IMAGE_NAME_CONTROLLER)
build_local_smartdocter_controller_image: IMAGEDIR := $(ROOT_DIR)/images/smartdocter-controller
build_local_smartdocter_controller_image: IMAGE_TAG := $(GIT_COMMIT_VERSION)
build_local_smartdocter_controller_image:
	@ $(BUILD_FINAL_IMAGE)


#---------

.PHONY: build_local_base_image
build_local_base_image: build_local_smartdocter_agent_base_image build_local_smartdocter_controller_base_image


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

.PHONY: build_local_smartdocter_agent_base_image
build_local_smartdocter_agent_base_image: IMAGEDIR := ./images/smartdocter-agent-base
build_local_smartdocter_agent_base_image: BASE_IMAGES := ${REGISTER}/${GIT_REPO}/smartdocter-agent-base
build_local_smartdocter_agent_base_image:
	@ $(BUILD_BASE_IMAGE)


.PHONY: build_local_smartdocter_controller_base_image
build_local_smartdocter_controller_base_image: IMAGEDIR := ./images/smartdocter-controller-base
build_local_smartdocter_controller_base_image: BASE_IMAGES := ${REGISTER}/${GIT_REPO}/smartdocter-controller-base
build_local_smartdocter_controller_base_image:
	@ $(BUILD_BASE_IMAGE)


# ==========================

.PHONY: package-charts
package-charts:
	@ make -C charts package

.PHONY: lint-golang
lint-golang: LINT_DIR := ./smartdocter-agent-cmd/... ./smartdocter-controller-cmd/...
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


.PHONY: lint-markdown-spell
lint-markdown-spell:
	if which mdspell &>/dev/null ; then \
  			mdspell  -r --en-us --ignore-numbers --target-relative .github/.spelling --ignore-acronyms  '**/*.md' '!vendor/**/*.md' ; \
  		else \
			$(CONTAINER_ENGINE) container run --rm \
				--entrypoint bash -v $(ROOT_DIR):/workdir  weizhoulan/spellcheck:latest  \
				-c "cd /workdir ; mdspell  -r --en-us --ignore-numbers --target-relative .github/.spelling --ignore-acronyms  '**/*.md' '!vendor/**/*.md' " ; \
  		fi

.PHONY: lint-markdown-spell-colour
lint-markdown-spell-colour:
	if which mdspell &>/dev/null ; then \
  			mdspell  -r --en-us --ignore-numbers --target-relative .github/.spelling --ignore-acronyms  '**/*.md' '!vendor/**/*.md' ; \
  		else \
			$(CONTAINER_ENGINE) container run --rm -it \
				--entrypoint bash -v $(ROOT_DIR):/workdir  weizhoulan/spellcheck:latest  \
				-c "cd /workdir ; mdspell  -r --en-us --ignore-numbers --target-relative .github/.spelling --ignore-acronyms  '**/*.md' '!vendor/**/*.md' " ; \
  		fi


.PHONY: lint-code-spell
lint-code-spell:
	$(QUIET) if ! which codespell &> /dev/null ; then \
  				echo "try to install codespell" ; \
  				if ! pip3 install codespell ; then \
  					echo "error, miss tool codespell, install it: pip3 install codespell" ; \
  					exit 1 ; \
  				fi \
  			fi ;\
  			codespell --config .github/codespell-config


.PHONY: fix-code-spell
fix-code-spell:
	$(QUIET) if ! which codespell &> /dev/null ; then \
  				echo "try to install codespell" ; \
  				if ! pip3 install codespell ; then \
  					echo "error, miss tool codespell, install it: pip3 install codespell" ; \
  					exit 1 ;\
  				fi \
  			fi; \
  			codespell --config .github/codespell-config  --write-changes



.PHONY: unitest-tests
unitest-tests: UNITEST_DIR := smartdocter-agent-cmd smartdocter-controller-cmd
unitest-tests:
	@echo "run unitest-tests"
	$(QUIET) $(ROOT_DIR)/tools/ginkgo.sh   \
		--cover --coverprofile=./coverage.out --covermode set  \
		--json-report unitestreport.json \
		-randomize-suites -randomize-all --keep-going  --timeout=1h  -p   --slow-spec-threshold=120s \
		-vv  -r   $(UNITEST_DIR)
	$(QUIET) go tool cover -html=./coverage.out -o coverage-all.html


# should label for each test file
.PHONY: check_test_label
check_test_label:
	@ALL_TEST_FILE=` find  ./  -name "*_test.go" -not -path "./vendor/*" ` ; FAIL="false" ; \
		for ITEM in $$ALL_TEST_FILE ; do \
			[[ "$$ITEM" == *_suite_test.go ]] && continue  ; \
			! grep 'Label(' $${ITEM} &>/dev/null && FAIL="true" && echo "error, miss Label in $${ITEM}" ; \
		done ; \
		[ "$$FAIL" == "true" ] && echo "error, label check fail" && exit 1 ; \
		echo "each test.go is labeled right"


.PHONY: e2e
e2e:
	@echo "run e2e"


.PHONY: preview_doc
preview_doc: PROJECT_DOC_DIR := ${ROOT_DIR}/docs
preview_doc:
	-docker stop doc_previewer &>/dev/null
	-docker rm doc_previewer &>/dev/null
	@echo "set up preview http server  "
	@echo "you can visit the website on browser with url 'http://127.0.0.1:8000' "
	[ -f "docs/mkdocs.yml" ] || { echo "error, miss docs/mkdocs.yml "; exit 1 ; }
	docker run --rm  -p 8000:8000 --name doc_previewer -v $(PROJECT_DOC_DIR):/host/docs \
        --entrypoint sh \
        --stop-timeout 3 \
        --stop-signal "SIGKILL" \
        squidfunk/mkdocs-material  -c "cd /host ; cp docs/mkdocs.yml ./ ;  mkdocs serve -a 0.0.0.0:8000"
	#sleep 10 ; if curl 127.0.0.1:8000 &>/dev/null  ; then echo "succeeded to set up preview server" ; else echo "error, failed to set up preview server" ; docker stop doc_previewer ; exit 1 ; fi


.PHONY: build_doc
build_doc: PROJECT_DOC_DIR := ${ROOT_DIR}/docs
build_doc: OUTPUT_TAR := site.tar.gz
build_doc:
	-docker stop doc_builder &>/dev/null
	-docker rm doc_builder &>/dev/null
	[ -f "docs/mkdocs.yml" ] || { echo "error, miss docs/mkdocs.yml "; exit 1 ; }
	-@ rm -f ./docs/$(OUTPUT_TAR)
	@echo "build doc html " ; \
		docker run --rm --name doc_builder  \
		-v ${PROJECT_DOC_DIR}:/host/docs \
        --entrypoint sh \
        squidfunk/mkdocs-material -c "cd /host ; cp ./docs/mkdocs.yml ./ ; mkdocs build ; cd site ; tar -czvf site.tar.gz * ; mv ${OUTPUT_TAR} ../docs/"
	@ [ -f "$(PROJECT_DOC_DIR)/$(OUTPUT_TAR)" ] || { echo "failed to build site to $(PROJECT_DOC_DIR)/$(OUTPUT_TAR) " ; exit 1 ; }
	@ echo "succeeded to build site to $(PROJECT_DOC_DIR)/$(OUTPUT_TAR) "



.PHONY: update-go-version
update-go-version: ## Update Go version for all the components
	@echo "GO_MAJOR_AND_MINOR_VERSION=${GO_MAJOR_AND_MINOR_VERSION}"
	@echo "GO_IMAGE_VERSION=${GO_IMAGE_VERSION}"
	# ===== Update Go version for GitHub workflow
	$(QUIET) for fl in $(shell find .github/workflows -name "*.yaml" -print) ; do \
  			sed -i 's/go-version: .*/go-version: ${GO_IMAGE_VERSION}/g' $$fl ; \
  			done
	@echo "Updated go version in GitHub Actions to $(GO_IMAGE_VERSION)"
	# ======= Update Go version in main.go.
	$(QUIET) for fl in $(shell find .  -name main.go -not -path "./vendor/*" -print); do \
		sed -i \
			-e 's|^//go:build go.*|//go:build go${GO_MAJOR_AND_MINOR_VERSION}|g' \
			-e 's|^// +build go.*|// +build go${GO_MAJOR_AND_MINOR_VERSION}|g' \
			$$fl ; \
	done
	# ====== Update Go version in go.mod
	$(QUIET) sed -i -E 's/^go .*/go '$(GO_MAJOR_AND_MINOR_VERSION)'/g' go.mod
	@echo "Updated go version in go.mod to $(GO_VERSION)"
ifeq (${shell [ -d ./test ] && echo done},done)
	# ======= Update Go version in test scripts.
	@echo "Updated go version in test scripts to $(GO_VERSION)"
endif
	# ===== Update Go version in Dockerfiles.
	$(QUIET) $(MAKE) -C images update-golang-image
	@echo "Updated go version in image Dockerfiles to $(GO_IMAGE_VERSION)"
