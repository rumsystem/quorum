package data

import (
)

type Activity struct{
    Context string `json:"@context"`
    Summary string `json:"summary"`
    Type string `json:"type"`
    Actor Actor `json:"actor"`
    Object Object `json:"object"`
}
