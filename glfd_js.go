package main

import "os"
import "fmt"
import "bytes"
import "bufio"
import "strings"
//import "strconv"
import "io/ioutil"
import "github.com/robertkrimen/otto"

//import "github.com/aebruno/twobit"
//import "github.com/abeconnelly/pasta"
//import "github.com/abeconnelly/pasta/gvcf"

import "github.com/abeconnelly/sloppyjson"

//import "reflect"

func info_otto(call otto.FunctionCall) otto.Value {
  v,e := otto.ToValue("ok info")
  if e!=nil { return otto.Value{} }
  return v
}

func (glfd *GLFD) tilesequence_otto(call otto.FunctionCall) otto.Value {
  tilepath,e := call.Argument(0).ToInteger()
  if e!=nil { return otto.Value{} }
  libver,e := call.Argument(1).ToInteger()
  if e!=nil { return otto.Value{} }
  tilestep,e := call.Argument(2).ToInteger()
  if e!=nil { return otto.Value{} }
  tilevarid,e := call.Argument(3).ToInteger()
  if e!=nil { return otto.Value{} }

  s,e := glfd.TileSequence(int(tilepath), int(libver), int(tilestep), int(tilevarid))
  if e!=nil { return otto.Value{} }

  v,e := otto.ToValue(s)
  return v
}

func align2pasta_otto(call otto.FunctionCall) otto.Value {
  refseq := call.Argument(0).String()
  altseq := call.Argument(1).String()

  s,e := AlignToPasta(refseq, altseq)
  if e!=nil { return otto.Value{} }

  v,e := otto.ToValue(s)
  if e!=nil { return otto.Value{} }
  return v
}

func align_otto(call otto.FunctionCall) otto.Value {
  refseq := call.Argument(0).String()
  altseq := call.Argument(1).String()

  ref_align, alt_align, score := align(refseq, altseq) ; _ = score

  //sa := []string{ref_align, alt_align}
  //v,e := otto.ToValue(sa)

  v,e := otto.ToValue(ref_align + "\n" + alt_align)
  if e!=nil { return otto.Value{} }
  return v
}


func emitgvcf_otto(call otto.FunctionCall) otto.Value {
  refseq := call.Argument(0).String()
  alt0seq := call.Argument(1).String()
  alt1seq := call.Argument(2).String()

  //DEBUG
  outs := bufio.NewWriter(os.Stdout)

  //EmitGVCF(refseq, alt0seq, alt1seq)
  EmitGVCF(outs, "unk", 0, refseq, alt0seq, alt1seq)

  v,e := otto.ToValue("ok")
  if e!=nil { return otto.Value{} }
  return v
}

//func (glfd *GLFD) tiletogvcf_x_otto(call otto.FunctionCall) otto.Value { }

func (glfd *GLFD) tiletogvcf_x_otto(call otto.FunctionCall) otto.Value {

  //fmt.Printf("tiletogvcf...\n")

  str := call.Argument(0).String()

  jso,e := sloppyjson.Loads(str)
  if e!=nil {  panic(e) }

  tilepath := int(jso.O["tilepath"].P)
  start_tilestep := int(jso.O["start_tilestep"].P)

  allele := [][]int{}
  allele = append(allele, []int{})
  allele = append(allele, []int{})

  n:=len(jso.O["allele"].L[0].L)
  for i:=0; i<n; i++ {
    allele[0] = append(allele[0], int(jso.O["allele"].L[0].L[i].P))
    allele[1] = append(allele[1], int(jso.O["allele"].L[1].L[i].P))
  }

  nocall := [][][]int{}
  nocall = append(nocall, [][]int{})
  nocall = append(nocall, [][]int{})
  for i:=0; i<n; i++ {
    nocall[0] = append(nocall[0], []int{})
    m:=len(jso.O["loq_info"].L[0].L[i].L)
    for j:=0; j<m; j++ {
      nocall[0][i] = append(nocall[0][i], int(jso.O["loq_info"].L[0].L[i].L[j].P))
    }

    nocall[1] = append(nocall[1], []int{})
    m=len(jso.O["loq_info"].L[1].L[i].L)
    for j:=0; j<m; j++ {
      nocall[1][i] = append(nocall[1][i], int(jso.O["loq_info"].L[1].L[i].L[j].P))
    }

  }

  ref_varid := []int{}
  for i:=0; i<len(allele[0]); i++ {
    //ref_varid = append(ref_varid, 0)
    ref_varid = append(ref_varid, glfd.RefV["hg19"][tilepath][start_tilestep+i])
  }

  _ = start_tilestep
  _ = allele
  _ = nocall

  bb := new(bytes.Buffer)
  outs := bufio.NewWriter(bb)

  s,e := glfd.TileToGVCF(outs, tilepath, 0, start_tilestep, allele, nocall, ref_varid) ; _ = s
  if e!=nil { panic(e) }

  json_gvcf_str := _to_json_gvcf(string(bb.Bytes()), "unk")

  v,e := otto.ToValue(json_gvcf_str)
  if e!=nil { return otto.Value{} }
  return v
}

func _to_json_gvcf(s, samp_name string) string {
  tot_res_a := []string{}
  lines := strings.Split(s,"\n")
  for i:=0; i<len(lines); i++ {
    parts := strings.Split(lines[i], "\t")

    res_a := []string{}

    if len(parts)<6 { continue; }

    t := fmt.Sprintf(`{ "chrom":"%s","pos":%s,"ref":"%s","alt":[`, parts[0], parts[1], parts[3])
    res_a = append(res_a, t)

    alt_parts := strings.Split(parts[4], ",")
    for j:=0; j<len(alt_parts); j++ {
      if j>0 {
        t = fmt.Sprintf(`,"%s"`, alt_parts[j])
      } else {
        t = fmt.Sprintf(`"%s"`, alt_parts[j])
      }
      res_a = append(res_a, t)
    }
    res_a = append(res_a, "],")


    res_a = append(res_a, `"format":[{`)
    res_a = append(res_a, fmt.Sprintf(`"sample-name":"%s","GT":"%s"`, samp_name, parts[9]))
    res_a = append(res_a, `}],`)

    res_a = append(res_a, `"info":{`)
    x_parts := strings.Split(parts[7], ";")
    for j:=0; j<len(x_parts); j++ {
      kv := strings.Split(x_parts[j], "=")
      if len(kv)==2 {
        if j>0 { res_a = append(res_a, ",") }

        if kv[0] == "END" {
          res_a = append(res_a, `"` + kv[0] + `":[` + kv[1] + `]`)
        } else {
          res_a = append(res_a, `"` + kv[0] + `":"` + kv[1] + `"`)
        }
      }
    }
    res_a = append(res_a, `}`)
    res_a = append(res_a, `}`)

    tot_res_a = append(tot_res_a, strings.Join(res_a, ""))
  }

  return `[` + strings.Join(tot_res_a, ",") + `]`
}

func (glfd *GLFD) JSVMRun(src string) (rstr string, e error) {
  js_vm := otto.New()

  fmt.Printf("JSVM_run:\n\n")

  init_js,err := ioutil.ReadFile("js/init.js")
  if err!=nil { e = err; return }
  js_vm.Run(init_js)

  js_vm.Set("info", info_otto)
  js_vm.Set("tilesequence", glfd.tilesequence_otto)
  js_vm.Set("aligntopasta", align2pasta_otto)
  js_vm.Set("align", align_otto)
  js_vm.Set("emitgvcf", emitgvcf_otto)
  //js_vm.Set("tiletogvcf", glfd.tiletogvcf_otto)
  js_vm.Set("tiletogvcf_x", glfd.tiletogvcf_x_otto)

  v,err := js_vm.Run(src)
  if err!=nil {
    e = err
    return
  }

  rstr,e = v.ToString()
  return
}
