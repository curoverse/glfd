package main

import "fmt"
import "os"
import "compress/gzip"
import "bufio"
import "strings"
import "strconv"
import "io/ioutil"

import "github.com/aebruno/twobit"
import "github.com/abeconnelly/sloppyjson"

func (glfd *GLFD) InitSpan(span_fn string) error {

  f,e := os.Open(span_fn)
  if e!=nil { return e }
  defer f.Close()

  gr,e := gzip.NewReader(f)
  if e!=nil { return e }
  defer gr.Close()

  glfd.TileLibSpan = make( map[int]map[int]map[int]int )

  scanner := bufio.NewScanner(gr)
  for scanner.Scan() {
    t := scanner.Text()
    if len(t)==0 { continue }
    kv := strings.Split(t,",")
    tileid_parts := strings.Split(kv[0], ".")
    span,e := strconv.Atoi(kv[1])
    if e!=nil { return e }

    p,e := strconv.ParseInt(tileid_parts[0], 16, 64)
    if e!=nil { return e}

    ver,e := strconv.ParseInt(tileid_parts[1], 16, 64) ; _ = ver
    if e!=nil { return e}

    s,e := strconv.ParseInt(tileid_parts[2], 16, 64)
    if e!=nil { return e}

    v,e := strconv.ParseInt(tileid_parts[3], 16, 64)
    if e!=nil { return e}

    if _,ok := glfd.TileLibSpan[int(p)] ; !ok {
      glfd.TileLibSpan[int(p)] = make( map[int]map[int]int )
    }

    if _,ok := glfd.TileLibSpan[int(p)][int(s)] ; !ok {
      glfd.TileLibSpan[int(p)][int(s)] = make( map[int]int )
    }

    glfd.TileLibSpan[int(p)][int(s)][int(v)] = span

  }

  return nil
}

func (glfd *GLFD) InitTagset(tagset_2bit_fn string) error {
  fp,err := os.Open(tagset_2bit_fn)
  if err!=nil { return err }
  defer fp.Close()

  tb,err := twobit.NewReader(fp)
  if err!=nil { return err }

  glfd.Tagset = make( map[int]string )

  names := tb.Names()
  for i:=0; i<len(names); i++ {
    parts := strings.Split(names[i], ".")
    p,e := strconv.ParseInt(parts[0], 16, 64)
    if e!=nil { return e }

    b,e := tb.Read(names[i])
    if e!=nil {return e}
    glfd.Tagset[int(p)] = string(b)
  }

  return nil
}

func (glfd *GLFD) InitHg19(hg19_json_fn string) error {

  s,e := ioutil.ReadFile(hg19_json_fn)
  if e!=nil { return e; }

  hgo,e := sloppyjson.Loads(string(s))
  if e!=nil { return e; }

  glfd.RefV = make( map[string]map[int][]int )
  glfd.RefV["hg19"] = make( map[int][]int )
  for k,_ := range hgo.O {
    tilepath := int(hgo.O[k].O["tilepath"].P)

    //DEBUG
    //fmt.Printf(">> %d (%x)\n", tilepath, tilepath)

    allele := []int{}
    for i:=0; i<len(hgo.O[k].O["allele"].L[0].L); i++ {
      allele = append(allele, int(hgo.O[k].O["allele"].L[0].L[i].P))
    }

    glfd.RefV["hg19"][tilepath] = allele
  }

  return nil
}

func (glfd *GLFD) InitAssembly(assembly_fn string) error {
  assembly := "hg19"

  glfd.Assembly = make( map[string]map[int]map[int]int )
  glfd.Assembly[assembly] = make( map[int]map[int]int )

  glfd.TilepathToChrom = make( map[int]string )

  f,e := os.Open(assembly_fn)
  if e!=nil { return e }
  defer f.Close()

  gr,e := gzip.NewReader(f)
  if e!=nil { return e }
  defer gr.Close()

  tilepath := -1

  scanner := bufio.NewScanner(gr)
  for scanner.Scan() {
    t := scanner.Text()
    if len(t)==0 { continue }
    if t[0] == '>' {
      parts := strings.Split(t[1:], ":")
      assembly = parts[0]
      chrom := parts[1]
      tilepath_str := parts[2]

      tilepath_i,e := strconv.ParseInt(tilepath_str, 16, 64)
      if e!=nil { return e }
      tilepath = int(tilepath_i)

      glfd.Assembly[assembly][tilepath] = make(map[int]int)
      glfd.TilepathToChrom[tilepath] = chrom
      continue
    }

    a_field_end := 0
    for i:=0; i<len(t); i++ {
      if t[i]==' ' || t[i] == '\t' { break; }
      a_field_end++
    }
    b_field_start := a_field_end
    for i:=a_field_end; i<len(t); i++ {
      if t[i]==' ' || t[i] == '\t' {
        b_field_start++;
        continue;
      }
      break;
    }

    //parts := strings.Split(t, "\t")
    //tilestep_i,e := strconv.ParseInt(parts[0], 16, 64)
    tilestep_i,e := strconv.ParseInt(t[0:a_field_end], 16, 64)
    if e!=nil { return e }
    tilestep := int(tilestep_i)

    //skip_space := 0
    //for i:=0; i<len(parts[1]); i++ {
    //  if parts[1][i] != ' ' {  break }
    //  skip_space++
    //}
    //end_refpos_i,e := strconv.ParseInt(parts[1][skip_space:], 10, 64)
    end_refpos_i,e := strconv.ParseInt(t[b_field_start:], 10, 64)
    if e!=nil { return e }
    end_refpos := int(end_refpos_i)

    glfd.Assembly[assembly][tilepath][tilestep] = end_refpos
  }

  // simple spot check
  fmt.Printf(">>>> %s %x %x (%s)-> %d\n", "hg19", 0x2fb, 0x102,
    glfd.TilepathToChrom[0x2fb],
    glfd.Assembly["hg19"][0x2fb][0x102])

  return nil
}

func (glfd *GLFD) InitCache() error {

  glfd.SeqCache = make(map[int]map[int]map[int]string)

  n:=len(glfd.RefV["hg19"]) ; _ = n

  //TESTING!!!!
  //test one cache path for now
  p := 0x2fb
  //for p:=0; p<n; p++ {

    //DEBUG
    fmt.Printf("caching %x\n", p)

    glfd.SeqCache[p] = make(map[int]map[int]string)

    m := len(glfd.Assembly["hg19"][p])
    for step:=0; step<m; step++ {
      glfd.SeqCache[p][step] = make(map[int]string)

      s,e := glfd.TileSequence(p, 0, step, 0)
      if e!=nil { return e }
      glfd.SeqCache[p][step][0] = s

      refvarid := glfd.RefV["hg19"][p][step]
      if refvarid != 0 {
        s,e = glfd.TileSequence(p, 0, step, refvarid)
        if e!=nil { return e }
        glfd.SeqCache[p][step][refvarid] = s
      }
    }
  //}

  return nil
}

func GLFDInit(glfd_dir, assembly_fn, tagset_fn, span_fn string) (*GLFD,error) {
  var glfd GLFD

  local_debug := true

  glfd.GLFDir = glfd_dir

  //---

  if local_debug { fmt.Printf("initializing hg19.json...\n") }

  e := glfd.InitHg19("js/hg19.json")
  if e!=nil { return nil, e }

  if local_debug { fmt.Printf("...done\n") }

  //---

  if local_debug { fmt.Printf("initializing assembly...\n") }

  e = glfd.InitAssembly(assembly_fn)
  if e!=nil { return nil, e }

  if local_debug { fmt.Printf("... done\n") }

  //---

  if local_debug { fmt.Printf("initializing tagset...\n") }

  e = glfd.InitTagset(tagset_fn)

  if local_debug { fmt.Printf("...done\n") }

  //---

  if local_debug { fmt.Printf("initializing span...\n") }

  e = glfd.InitSpan(span_fn)

  if local_debug { fmt.Printf("...done\n") }


  //if local_debug { fmt.Printf("initalizing cache...\n") }

  //e = glfd.InitCache()
  //if e!=nil { return nil, e }

  //if local_debug { fmt.Printf("...done\n") }

  return &glfd,nil
}