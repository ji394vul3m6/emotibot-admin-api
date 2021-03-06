-- MySQL Script generated by MySQL Workbench
-- Mon Oct 30 18:07:52 2017
-- Model: New Model    Version: 1.0
-- MySQL Workbench Forward Engineering

SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;
SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;
SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='TRADITIONAL,ALLOW_INVALID_DATES';

-- -----------------------------------------------------
-- Schema voice_emotion
-- -----------------------------------------------------

-- -----------------------------------------------------
-- Schema voice_emotion
-- -----------------------------------------------------
CREATE SCHEMA IF NOT EXISTS `voice_emotion` DEFAULT CHARACTER SET utf8 ;
USE `voice_emotion` ;

-- -----------------------------------------------------
-- Table `voice_emotion`.`fileInformation`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `voice_emotion`.`fileInformation` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `file_id` VARCHAR(48) NOT NULL,
  `path` VARCHAR(256) NOT NULL,
  `file_name` VARCHAR(256) NULL,
  `file_type` VARCHAR(8) NULL,
  `size` INT UNSIGNED NULL,
  `duration` INT UNSIGNED NULL,
  `created_time` BIGINT UNSIGNED NULL,
  `checksum` VARCHAR(32) NULL DEFAULT '',
  `priority` SMALLINT UNSIGNED NULL DEFAULT 0,
  `appid` VARCHAR(32) NOT NULL,
  `analysis_start_time` BIGINT UNSIGNED NULL DEFAULT 0,
  `analysis_end_time` BIGINT UNSIGNED NULL DEFAULT 0,
  `analysis_result` INT NULL DEFAULT -1,
  `upload_time` BIGINT UNSIGNED NOT NULL,
  `real_duration` INT NULL DEFAULT -1,
  PRIMARY KEY (`id`),
  INDEX `create_time` (`created_time` ASC),
  UNIQUE INDEX `file_id_UNIQUE` (`file_id` ASC),
  INDEX `appid` (`appid` ASC),
  INDEX `analysis_result` (`analysis_result` ASC),
  UNIQUE INDEX `id_UNIQUE` (`id` ASC))
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `voice_emotion`.`analysisInformation`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `voice_emotion`.`analysisInformation` (
  `segment_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `id` BIGINT UNSIGNED NOT NULL,
  `segment_start_time` FLOAT NULL,
  `segment_end_time` FLOAT NULL,
  `channel` INT UNSIGNED NULL,
  `status` INT NULL,
  `extra_info` BLOB NULL,
  PRIMARY KEY (`segment_id`),
  INDEX `id` (`id` ASC),
  UNIQUE INDEX `unique_compose` (`id` ASC, `segment_start_time` ASC, `segment_end_time` ASC, `channel` ASC))
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `voice_emotion`.`emotionInformation`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `voice_emotion`.`emotionInformation` (
  `emoInfo_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `segment_id` BIGINT UNSIGNED NOT NULL,
  `emotion_type` INT NULL,
  `score` FLOAT NULL,
  INDEX `segment_id` (`segment_id` ASC),
  PRIMARY KEY (`emoInfo_id`))
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `voice_emotion`.`channelScore`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `voice_emotion`.`channelScore` (
  `chanScore_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `id` BIGINT UNSIGNED NOT NULL,
  `channel` INT UNSIGNED NOT NULL,
  `emotion_type` INT UNSIGNED NOT NULL,
  `score` FLOAT UNSIGNED NOT NULL,
  INDEX `score` (`score` ASC),
  INDEX `id` (`id` ASC),
  PRIMARY KEY (`chanScore_id`),
  INDEX `channel` (`channel` ASC),
  INDEX `emotion_type` (`emotion_type` ASC),
  UNIQUE INDEX `unique_compose` (`id` ASC, `channel` ASC, `emotion_type` ASC))
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `voice_emotion`.`userDefinedTags`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `voice_emotion`.`userDefinedTags` (
  `defined_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `id` BIGINT UNSIGNED NULL,
  `tag` VARCHAR(64) NOT NULL,
  PRIMARY KEY (`defined_id`),
  INDEX `id` (`id` ASC),
  INDEX `tag` (`tag` ASC),
  UNIQUE INDEX `unique_tag` (`id` ASC, `tag` ASC))
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `voice_emotion`.`reportTo`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `voice_emotion`.`reportTo` (
  `report_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `crontab` VARCHAR(32) NOT NULL,
  `email` VARCHAR(1024) NOT NULL,
  `appid` VARCHAR(32) NOT NULL,
  PRIMARY KEY (`report_id`),
  INDEX `appid` (`appid` ASC))
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `voice_emotion`.`userColumn`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `voice_emotion`.`userColumn` (
  `col_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `col_type` SMALLINT UNSIGNED NOT NULL,
  `col_name` VARCHAR(64) NOT NULL,
  `appid` VARCHAR(32) NOT NULL,
  `default_value` VARCHAR(256) NOT NULL,
  PRIMARY KEY (`col_id`),
  INDEX `index` (`appid` ASC),
  UNIQUE INDEX `col_id_UNIQUE` (`col_id` ASC))
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `voice_emotion`.`userColumnValue`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `voice_emotion`.`userColumnValue` (
  `col_val_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `id` BIGINT UNSIGNED NULL DEFAULT NULL,
  `col_id` BIGINT UNSIGNED NULL DEFAULT NULL,
  `col_value` VARCHAR(256) NOT NULL,
  PRIMARY KEY (`col_val_id`),
  UNIQUE INDEX `col_val_id_UNIQUE` (`col_val_id` ASC),
  UNIQUE INDEX `unique` (`id` ASC, `col_id` ASC),
  INDEX `file_index` (`id` ASC),
  INDEX `col_index` (`col_id` ASC),
  INDEX `col_value_index` (`col_value` ASC))
ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `voice_emotion`.`userSelectableValue`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `voice_emotion`.`userSelectableValue` (
  `sel_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `col_id` BIGINT UNSIGNED NULL DEFAULT NULL,
  `sel_value` VARCHAR(256) NOT NULL,
  PRIMARY KEY (`sel_id`),
  UNIQUE INDEX `sel_id_UNIQUE` (`sel_id` ASC),
  INDEX `col_index` (`col_id` ASC),
  UNIQUE INDEX `unique_sel` (`col_id` ASC, `sel_value` ASC))
ENGINE = InnoDB;


SET SQL_MODE=@OLD_SQL_MODE;
SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;
SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;
