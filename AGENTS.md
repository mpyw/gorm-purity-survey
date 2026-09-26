# Repository instructions

gorm-purity-survey measures changes in GORM `*gorm.DB` behavior across patch versions. It is a survey and documentation project. The method categories, version matrix, Docker strategy, and test methodology are in [implementation notes](design/implementation.md). Read the relevant section before changing measurements or conclusions.

Keep reported findings tied to the version and evidence that produced them. Run the relevant Go tests and survey checks for the area changed; the commands and Docker workflow are described in the implementation notes.
