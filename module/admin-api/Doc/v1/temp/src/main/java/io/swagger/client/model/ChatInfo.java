/*
 * 机器人设置API
 * This is api document page for robot setting RestAPIs
 *
 * OpenAPI spec version: 1.0.0
 * Contact: danielwu@emotibot.com
 *
 * NOTE: This class is auto generated by the swagger code generator program.
 * https://github.com/swagger-api/swagger-codegen.git
 * Do not edit the class manually.
 */


package io.swagger.client.model;

import java.util.Objects;
import com.google.gson.TypeAdapter;
import com.google.gson.annotations.JsonAdapter;
import com.google.gson.annotations.SerializedName;
import com.google.gson.stream.JsonReader;
import com.google.gson.stream.JsonWriter;
import io.swagger.annotations.ApiModel;
import io.swagger.annotations.ApiModelProperty;
import java.io.IOException;
import java.util.ArrayList;
import java.util.List;

/**
 * ChatInfo
 */
@javax.annotation.Generated(value = "io.swagger.codegen.languages.java.JavaClientCodegen", date = "2018-03-01T18:34:36.180+08:00")
public class ChatInfo {
@SerializedName("type")
  private Integer type = null;
  @SerializedName("contents")
  private List<String> contents = null;
  
  public ChatInfo type(Integer type) {
    this.type = type;
    return this;
  }

  
  /**
  * Get type
  * @return type
  **/
  @ApiModelProperty(value = "")
  public Integer getType() {
    return type;
  }
  public void setType(Integer type) {
    this.type = type;
  }
  
  public ChatInfo contents(List<String> contents) {
    this.contents = contents;
    return this;
  }

  public ChatInfo addContentsItem(String contentsItem) {
    
    if (this.contents == null) {
      this.contents = new ArrayList<String>();
    }
    
    this.contents.add(contentsItem);
    return this;
  }
  
  /**
  * Get contents
  * @return contents
  **/
  @ApiModelProperty(example = "[\"话术1\",\"话术2\"]", value = "")
  public List<String> getContents() {
    return contents;
  }
  public void setContents(List<String> contents) {
    this.contents = contents;
  }
  
  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ChatInfo chatInfo = (ChatInfo) o;
    return Objects.equals(this.type, chatInfo.type) &&
        Objects.equals(this.contents, chatInfo.contents);
  }

  @Override
  public int hashCode() {
    return Objects.hash(type, contents);
  }
  
  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ChatInfo {\n");
    
    sb.append("    type: ").append(toIndentedString(type)).append("\n");
    sb.append("    contents: ").append(toIndentedString(contents)).append("\n");
    sb.append("}");
    return sb.toString();
  }

  /**
   * Convert the given object to string with each line indented by 4 spaces
   * (except the first line).
   */
  private String toIndentedString(java.lang.Object o) {
    if (o == null) {
      return "null";
    }
    return o.toString().replace("\n", "\n    ");
  }

  
}


