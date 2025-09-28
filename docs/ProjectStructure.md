# Project Structure Specification

# Project Structure

## Project Directories/Root Folders

For full Klar projects with a glas.pack, these folders inside the same folder as glas.pack should be detected. Any other folder is ignored by the Klar toolchain. Modules are found in these folders.

|    Name     | Contains                                                             |         Public         | Downloaded |
| :---------: | -------------------------------------------------------------------- | :--------------------: | :--------: |
|    `cmd`    | Installable and runnable commands. Even if your project is private, it is recommended to place your commands in this folder. |          Yes           |    Yes     |
|   `dist`    | Build output. This should be included in `.gitignore`.               |           No           |     No     |
|   `docs`    | Markdown files that can be included in package documentation.        |           No           |    Yes     |
| `external`  | Files written in other languages for linking in `@external`. Available to all packages. | No  |    Yes     |
| `generated` | Generated files, usually Klar scripts or external scripts. Files here should not be manually edited. |           No           |    Yes     |
|    `pkg`    | Packages in a single project/monorepo                                | Installed individually |            |
|  `recipes`  | Klar recipes (coming soon). Individual Klar scripts are placed here. |           No           |     No     |
|  `scripts`  | Individual Klar scripts not inside modules. Used for development.    |           No           |     No     |
|  `shared`   | Modules available to all **packages** in a project, but not outside. |           No           |    Yes     |
|    `src`    | Entry files for a package. Most files and modules should be here.    |          Yes           |    Yes     |
|   `.klar`   | Project-specific folder for project data, cache, and dependencies. This should be included in `.gitignore`. |           No           |     No     |

## Project-less Scripts

Klar scripts placed outside a project with a glas.pack file are allowed.

-   Are discrete modules, even with files in the same folder
-   Must be run by name
-   Cannot install or import modules outside the standard library

# Modules

Modules are defined by directories. Creating a directory inside a directory creates a submodule. Modules can be imported into Klar scripts by their path. Klar import paths are separated by dot characters (`.`)

# Module Identifiers

The module identifier is the name of the directory. A module identifier:

-   can be any Klar identifier
-   can contain any Unicode letter or digit or underscores (`_`)
-   cannot be a keyword (such as `import, func,` or `go`) or a modifier (`public, opaque`, etc.)
-   cannot begin with a digit
-   cannot be only a single underscore (`_`)

**Allowed:** a, b6, core, HOLA  
**Not allowed:** import, go, public, 67, 1char, \_  
Important: Validation of directory names is not required. It is recommended that users avoid module names that are OS-reserved, such as `con` on Windows.

## Path Limits

No more than four (4) folders/parts in a module. This starts from the folder inside `src`.  
**Allowed:** `a.b.c.d` (4 parts)  
**Not allowed:** `a.b.c.d.e` (5 parts)

# Installing Modules

Files included in a downloaded module: glas.pack, glas.lock, klardoc.json

# Assets

**Scripts:** .klar, .js, .ts, .wasm, .php  
**Stylesheets:** .css, .scss, .sass, .less
**Media:** .png, .jp(e)g, .svg, .webp, .gif, .mp4, .mp3, .wav, .webm, .tiff  
**Markup:** .xml, .json, .html, .klarml, .txt, .csv, .tsv  
**Fonts:** .ttf, .otf, .woff, .woff2
