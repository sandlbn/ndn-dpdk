#!/bin/bash
set -e
set -o pipefail
cd "$( dirname "${BASH_SOURCE[0]}" )"

(
  echo '#ifndef NDN_DPDK_NDN_TLV_TYPE_H'
  echo '#define NDN_DPDK_NDN_TLV_TYPE_H'
  echo
  echo '/// \file'
  echo
  echo 'typedef enum TlvType {'
  awk  'NF==2 { print "  TT_" $1 " = 0x" $2 "," }' tlv-type.tsv
  echo '} TlvType;'
  echo
  echo '#endif // NDN_DPDK_NDN_TLV_TYPE_H'
) > tlv-type.h

(
  echo 'package ndn'
  echo
  echo 'import "fmt"'
  echo
  echo 'type TlvType uint32'
  echo
  echo 'const ('
  awk 'NF==2 { print "TT_" $1 " TlvType = 0x" $2  }' tlv-type.tsv
  echo ')'
  echo
  echo 'func (tt TlvType) String() string {'
  echo '  switch tt {'
  awk  'NF==2 {
          if (!numberToType[$2]) {
            numberToType[$2] = $1;
            print "  case TT_" $1 ": return \"" $1 "\""
          } else {
            print "  // TT_" $1 " has same number as " numberToType[$2]
          }
        }' tlv-type.tsv
  echo '  }'
  echo '  return fmt.Sprintf("%d", tt)'
  echo '}'
) | gofmt -s > tlv-type.go

(
  echo 'export enum TT {'
  awk  'NF==2 { print "  " $1 " = 0x" $2 "," }' tlv-type.tsv
  echo '}'
) > tlv-type.ts
