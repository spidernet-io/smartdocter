include  ../Makefile.defs

ALL_CHART := smartdocter

.PHONY: all
all: package


.PHONY: lint
lint:
	@ for ITEM in $(ALL_CHART) ; do \
   		echo "check $${ITEM}" ; \
   		helm lint --with-subcharts $${ITEM} ; \
   		done

.PHONY: package
package: lint
	-@rm -rf $(CHART_OUTPUT)
	-@mkdir -p $(CHART_OUTPUT)
	@ for ITEM in $(ALL_CHART) ; do \
   		echo "package $${ITEM}" ; \
   		helm package $${ITEM} ; \
   		done
	@ mv *.tgz $(CHART_OUTPUT)


.PHONY: clean
clean:
	-@rm -f *.tgz
	-@rm -rf $(CHART_OUTPUT)
