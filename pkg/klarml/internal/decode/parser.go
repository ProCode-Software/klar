package decode

func (d *Decoder) Curr() byte {
	return d.Buffer[d.Pos]
}

func (d *Decoder) HasBytes() bool {
	return d.Pos < len(d.Buffer)-1
}

func (d *Decoder) Advance() byte {
	b := d.Buffer[d.Pos]
	d.Pos++
	if !d.HasBytes() {
		
	}
	return b
}