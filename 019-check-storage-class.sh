#!/bin/bash

SELECT_ENV_FILE="000-env.sh"
if [[ -f "`dirname $0`/${SELECT_ENV_FILE}" ]]; then
	source "`dirname $0`/${SELECT_ENV_FILE}"
else 
	echo "Environment file '${SELECT_ENV_FILE}' not found"
	exit 1
fi

VOL_BIND_MODE=`${KUBE} get sc ${SOLBK_STORAGECLASS} -o custom-columns=MODE:.volumeBindingMode --no-headers 2> /dev/null`
VOL_EXPANSION=`${KUBE} get sc ${SOLBK_STORAGECLASS} -o custom-columns=MODE:.allowVolumeExpansion --no-headers 2> /dev/null`

echo -n "Storage Class '${SOLBK_STORAGECLASS}' checks:
  VolumeBindingMode: ${VOL_BIND_MODE} ["
[[ "${VOL_BIND_MODE}" != "WaitForFirstConsumer" ]] && echo -n "NOT " && SCERR=true
echo -n "OK]
  AllowVolumeExpansion: ${VOL_EXPANSION} ["
[[ "${VOL_EXPANSION}" != "true" ]] && echo -n "NOT " && SCERR=true
echo "OK]"; echo ""
if [[ -z "${SCERR}" ]]; then
       	echo "All checks OK."
else
	echo "Error. SC checks failed."
	exit 1
fi

